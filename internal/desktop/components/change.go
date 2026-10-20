package components

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"whaleisle/internal/store/storelocale"
)

// BridgeChange holds a verified, immutable staged version. Preparing a change
// never stops a running instance, rewrites installed.json, or touches user data.
// The caller must use the Host task-admission guard around Apply.
type BridgeChange struct {
	root, dshHome, version string
	from                   installedFile
}

func (p *BridgeChange) Version() string { return p.version }

func PrepareWhaleBridgeUpdate(ctx context.Context, userData, route string) (*BridgeChange, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeSupportsWindowsX64Only))
	}
	endpoint, err := bridgeReleaseEndpoint(route)
	if err != nil {
		return nil, err
	}
	return prepareBridgeUpdate(ctx, userData, endpoint, &http.Client{Timeout: 5 * time.Minute}, false)
}

func prepareBridgeUpdate(ctx context.Context, userData, endpoint string, client *http.Client, fixture bool) (*BridgeChange, error) {
	if !componentInstallMu.TryLock() {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsAComponentInstallOrSwitchIsAlreadyRun))
	}
	defer componentInstallMu.Unlock()
	p, err := newBridgeChange(userData)
	if err != nil {
		return nil, err
	}
	m, asset, err := resolveBridgeRelease(ctx, endpoint, client, fixture)
	if err != nil {
		return nil, err
	}
	if compareVersions(m.Version, p.from.Version) <= 0 {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsNoNewerVersionOnThisChannelUseRollbac))
	}
	if _, err := stageBridgeRelease(ctx, p.root, m, asset, client, fixture); err != nil {
		return nil, err
	}
	p.version = m.Version
	return p, nil
}

func PrepareWhaleBridgeRollback(userData string) (*BridgeChange, error) {
	if !componentInstallMu.TryLock() {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsAComponentInstallOrSwitchIsAlreadyRun))
	}
	defer componentInstallMu.Unlock()
	p, err := newBridgeChange(userData)
	if err != nil {
		return nil, err
	}
	if !bridgeVersion.MatchString(p.from.Previous) || p.from.Previous == p.from.Version {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsNoVersionToRollBackTo))
	}
	if _, err := verifyBridgeVersion(p.root, p.from.Previous); err != nil {
		return nil, err
	}
	p.version = p.from.Previous
	return p, nil
}

func newBridgeChange(userData string) (*BridgeChange, error) {
	root := filepath.Join(userData, "components", "whalebridge")
	inst, err := checkedBridgeInstalled(root)
	if err != nil {
		return nil, err
	}
	// A recoverable old payload is required before accepting a replacement.
	if _, err := verifyBridgeVersion(root, inst.Version); err != nil {
		return nil, err
	}
	return &BridgeChange{root: root, dshHome: filepath.Join(userData, "dsh-home"), from: inst}, nil
}

func checkedBridgeInstalled(root string) (installedFile, error) {
	var inst installedFile
	b, err := os.ReadFile(filepath.Join(root, "installed.json"))
	if os.IsNotExist(err) {
		return inst, ErrNotInstalled
	}
	if err != nil {
		return inst, err
	}
	if json.Unmarshal(b, &inst) != nil || !bridgeVersion.MatchString(inst.Version) {
		return inst, errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeInstallRecordExisting))
	}
	return inst, nil
}

// verifyBridgeVersion also protects rollback and cached staged downloads: neither
// a symlink nor a changed executable may be promoted using only a version name.
func verifyBridgeVersion(root, version string) (bridgeReleaseManifest, error) {
	var m bridgeReleaseManifest
	if !bridgeVersion.MatchString(version) {
		return m, errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeVersion))
	}
	dir := filepath.Join(root, "versions", version)
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return m, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeVersionDirectoryIsMissing))
	}
	for _, name := range []string{bridgeManifest, bridgeAsset} {
		if info, err := os.Lstat(filepath.Join(dir, name)); err != nil || !info.Mode().IsRegular() {
			return m, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeVersionFileIsMissingOrI))
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, bridgeManifest))
	if err != nil {
		return m, err
	}
	if json.Unmarshal(b, &m) != nil || validateBridgeManifest(m) != nil || m.Version != version {
		return m, errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeLocalVersionManifest))
	}
	spec := m.Platforms["win32-x64"]
	f, err := os.Open(filepath.Join(dir, bridgeAsset))
	if err != nil {
		return m, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, spec.Size+1))
	if err != nil || n != spec.Size || fmt.Sprintf("%x", h.Sum(nil)) != spec.SHA256 {
		return m, errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeLocalProgramSizeOrSha256Ver))
	}
	return m, nil
}

type bridgeProcess interface {
	Terminate(context.Context) error // Must confirm exit of the pinned process handle.
	Close()
}
type bridgeLifecycle struct {
	client *http.Client
	open   func(int, string) (bridgeProcess, error)
	start  func(string, string) error
}

func bridgeRuntime() bridgeLifecycle {
	return bridgeLifecycle{client: &http.Client{Timeout: 2 * time.Second}, open: openBridgeProcess, start: startWhalebridgeLocked}
}

func (p *BridgeChange) Apply(ctx context.Context) error { return p.apply(ctx, bridgeRuntime()) }

func (p *BridgeChange) apply(ctx context.Context, life bridgeLifecycle) error {
	if !componentInstallMu.TryLock() {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsAComponentInstallOrSwitchIsAlreadyRun))
	}
	defer componentInstallMu.Unlock()
	if !bridgeStartMu.TryLock() {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeIsStartingOrStopping))
	}
	defer bridgeStartMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := checkedBridgeInstalled(p.root)
	if err != nil {
		return err
	}
	if current != p.from {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeVersionChangedCheckForUp))
	}
	for _, version := range []string{p.version, current.Version} {
		if _, err := verifyBridgeVersion(p.root, version); err != nil {
			return err
		}
	}
	running, err := stopBridgeInstance(ctx, p.root, current.Version, life)
	if err != nil {
		return err
	}
	next := installedFile{Version: p.version, Previous: current.Version}
	if err := replaceBridgeRecord(p.root, next); err != nil {
		return p.restore(current, running, life, err)
	}
	// Updating a stopped installation must not silently launch a credentialed service.
	if !running {
		return nil
	}
	if err := life.start(p.root, p.dshHome); err != nil {
		return p.restore(current, true, life, err)
	}
	return nil
}

func (p *BridgeChange) restore(old installedFile, running bool, life bridgeLifecycle, cause error) error {
	// A failed launch cleans up only its own process. If its exit is uncertain or a
	// different instance appeared, never overwrite that owner's record or launch a duplicate.
	if _, err := os.Lstat(filepath.Join(p.root, "data", "state.json")); errors.Is(cause, errBridgeExitUnconfirmed) || !os.IsNotExist(err) {
		return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheVersionSwitchDidNotFinishAndTheIn), cause))
	}
	if err := replaceBridgeRecord(p.root, old); err != nil {
		return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheVersionSwitchFailedAndTheInstallRe), err))
	}
	if running {
		if err := life.start(p.root, p.dshHome); err != nil {
			return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheVersionSwitchFailedTheOldVersionRe), err))
		}
	}
	return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheVersionSwitchDidNotTakeEffectTheP), cause))
}

func replaceBridgeRecord(root string, inst installedFile) error {
	b, _ := json.Marshal(inst)
	f, err := os.CreateTemp(root, ".installed-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(root, "installed.json"))
}

func stopBridgeInstance(ctx context.Context, root, version string, life bridgeLifecycle) (bool, error) {
	path := filepath.Join(root, "data", "state.json")
	st, err := readBridgeState(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Pin an OS process handle and check its exact installed image before status
	// authentication and maintenance. No PID-only kill, even after launcher restart.
	if st.ID != "whalebridge" || st.PID <= 0 || st.Version != version {
		return false, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheExistingWhalebridgeInstanceIdentityD))
	}
	proc, err := life.open(st.PID, filepath.Join(root, "versions", version, bridgeAsset))
	if err != nil {
		return false, err
	}
	defer proc.Close()
	if err := verifyBridgeReady(ctx, st, version, st.PID, life.client); err != nil {
		return false, err
	}
	if err := bridgeMaintenance(ctx, st, true, life.client); err != nil {
		_ = bridgeMaintenance(context.Background(), st, false, life.client)
		return false, err
	}
	stopped := false
	defer func() {
		if !stopped {
			_ = bridgeMaintenance(context.Background(), st, false, life.client)
		}
	}()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	// Re-read immediately before termination: a replacement state is never ours.
	if current, err := readBridgeState(path); err != nil || current != st {
		return false, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeInstanceChangedSoTheSwit))
	}
	if err := proc.Terminate(ctx); err != nil {
		return false, err
	}
	stopped = true
	if current, err := readBridgeState(path); err == nil && current == st {
		if err := os.Remove(path); err != nil {
			return true, err
		}
	} else if !os.IsNotExist(err) {
		return true, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeStateChangedAfterStopping))
	}
	return true, nil
}
