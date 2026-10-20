package components


import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"whaleisle/internal/store/storelocale"
)

// BridgeUninstall pins the verified installed version and default-model impact
// shown in the confirmation. Apply retains data; explicit removal only targets
// this component's data directory. The caller holds Host task admission protection.
type BridgeUninstall struct {
	root, dshHome string
	from          installedFile
	defaultModel  bool
}

func (p *BridgeUninstall) Version() string       { return p.from.Version }
func (p *BridgeUninstall) DefaultModel() bool    { return p.defaultModel }
func (p *BridgeUninstall) DataDirectory() string { return filepath.Join(p.root, "data") }

func WhaleBridgeManagementSupported() bool {
	return runtime.GOOS == "windows" && runtime.GOARCH == "amd64"
}

type bridgeRemovalOps struct {
	inspect    func(context.Context, string, string, string) (bool, error)
	stop       func(context.Context, string, string) (bool, error)
	start      func(string, string) error
	disconnect func(context.Context, string, string, string) error
	rename     func(string, string) error
	remove     func(string) error
}

func bridgeRemovalRuntime() bridgeRemovalOps {
	life := bridgeRuntime()
	return bridgeRemovalOps{
		inspect: inspectBridgeRemoval,
		stop: func(ctx context.Context, root, version string) (bool, error) {
			return stopBridgeInstance(ctx, root, version, life)
		},
		start: life.start,
		disconnect: func(ctx context.Context, exe, data, dshHome string) error {
			_, err := bridgeOfflineCommand(ctx, exe, data, dshHome, "whalebridge-disconnect")
			return err
		},
		rename: os.Rename, remove: os.RemoveAll,
	}
}

func PrepareWhaleBridgeUninstall(ctx context.Context, userData string) (*BridgeUninstall, error) {
	if !WhaleBridgeManagementSupported() {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeComponentManagementRequiresWi))
	}
	return prepareBridgeUninstall(ctx, userData, bridgeRemovalRuntime())
}

func prepareBridgeUninstall(ctx context.Context, userData string, ops bridgeRemovalOps) (*BridgeUninstall, error) {
	if !componentInstallMu.TryLock() {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsAComponentInstallOrSwitchIsAlreadyRun))
	}
	defer componentInstallMu.Unlock()
	if !bridgeStartMu.TryLock() {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeIsStartingOrStopping))
	}
	defer bridgeStartMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := filepath.Join(userData, "components", "whalebridge")
	if err := validateBridgeRemovalPaths(root); err != nil {
		return nil, err
	}
	change, err := newBridgeChange(userData)
	if err != nil {
		return nil, err
	}
	impact, err := ops.inspect(ctx, root, change.dshHome, change.from.Version)
	if err != nil {
		return nil, err
	}
	return &BridgeUninstall{root: root, dshHome: change.dshHome, from: change.from, defaultModel: impact}, nil
}

// Apply rechecks the confirmation, preserves runtime ownership and quarantines
// only payloads/installation metadata until the existing atomic disconnect CLI
// succeeds. The default Apply entry keeps provider data, accounts and usage.
// Chats and other channels are outside the optional component-data cleanup.
// The official disconnect command changes its owned integration settings and
// gateway credential reference, and newer versions unwind managed agents too.
func (p *BridgeUninstall) Apply(ctx context.Context) (string, error) {
	return p.ApplyWithDataRemoval(ctx, false)
}

// ApplyWithDataRemoval matches the original removeData choice. Pass true only
// after the user explicitly confirms deleting this component's configuration.
func (p *BridgeUninstall) ApplyWithDataRemoval(ctx context.Context, removeData bool) (string, error) {
	if !WhaleBridgeManagementSupported() {
		return "", errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeComponentManagementRequiresWi))
	}
	return p.applyWithDataRemoval(ctx, bridgeRemovalRuntime(), removeData)
}

func (p *BridgeUninstall) apply(ctx context.Context, ops bridgeRemovalOps) (string, error) {
	return p.applyWithDataRemoval(ctx, ops, false)
}

func (p *BridgeUninstall) applyWithDataRemoval(ctx context.Context, ops bridgeRemovalOps, removeData bool) (string, error) {
	if !componentInstallMu.TryLock() {
		return "", errors.New(storelocale.T(storelocale.KeyStoreComponentsAComponentInstallOrSwitchIsAlreadyRun))
	}
	defer componentInstallMu.Unlock()
	if !bridgeStartMu.TryLock() {
		return "", errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeIsStartingOrStopping))
	}
	defer bridgeStartMu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := validateBridgeRemovalPaths(p.root); err != nil {
		return "", err
	}
	current, err := checkedBridgeInstalled(p.root)
	if err != nil {
		return "", err
	}
	if current != p.from {
		return "", errors.New(storelocale.T(storelocale.KeyStoreComponentsTheInstalledWhalebridgeVersionChangedCo))
	}
	if _, err := verifyBridgeVersion(p.root, p.Version()); err != nil {
		return "", err
	}
	impact, err := ops.inspect(ctx, p.root, p.dshHome, p.Version())
	if err != nil {
		return "", err
	}
	if impact != p.defaultModel {
		return "", errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeDefaultModelStateChanged))
	}
	running, err := ops.stop(ctx, p.root, p.Version())
	if err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(p.root, ".uninstall-")
	if err != nil {
		return "", p.restoreRemoval("", false, false, running, ops)
	}
	versions := filepath.Join(p.root, "versions")
	record := filepath.Join(p.root, "installed.json")
	if err := ops.rename(versions, filepath.Join(stage, "versions")); err != nil {
		return "", p.restoreRemoval(stage, false, false, running, ops)
	}
	if err := ops.rename(record, filepath.Join(stage, "installed.json")); err != nil {
		return "", p.restoreRemoval(stage, true, false, running, ops)
	}
	// Verify again at the quarantined path immediately before executing it.
	if _, err := verifyBridgeVersion(stage, p.Version()); err != nil {
		return "", p.restoreRemoval(stage, true, true, running, ops)
	}
	if _, err := os.Lstat(filepath.Join(p.root, "data", "state.json")); !os.IsNotExist(err) {
		return "", p.restoreRemoval(stage, true, true, running, ops)
	}
	if err := ops.disconnect(ctx, filepath.Join(stage, "versions", p.Version(), bridgeAsset), filepath.Join(p.root, "data"), p.dshHome); err != nil {
		return "", p.restoreRemoval(stage, true, true, running, ops)
	}
	if err := ops.remove(stage); err != nil {
		return storelocale.T(storelocale.KeyStoreComponentsWhalebridgeWasUninstalledAndDisconnected), nil
	}
	if removeData {
		if err := ops.remove(p.DataDirectory()); err != nil {
			return storelocale.T(storelocale.KeyStoreComponentsWhalebridgeWasUninstalledAndDisconnected2) + err.Error() + storelocale.T(storelocale.KeyStoreComponentsChatHistoryAndOtherChannelsWereNotDel), nil
		}
	}
	return "", nil
}

// Recovery never overwrites a replacement owner or starts a duplicate process.
// On any uncertain recovery the quarantine is retained rather than destroyed.
func (p *BridgeUninstall) restoreRemoval(stage string, movedVersions, movedRecord, running bool, ops bridgeRemovalOps) error {
	retained := errors.New(storelocale.T(storelocale.KeyStoreComponentsTheUninstallAndAutomaticRecoveryDidNot))
	if _, err := os.Lstat(filepath.Join(p.root, "data", "state.json")); !os.IsNotExist(err) {
		return retained
	}
	if movedVersions {
		if _, err := os.Lstat(filepath.Join(p.root, "versions")); !os.IsNotExist(err) {
			return retained
		}
		if err := ops.rename(filepath.Join(stage, "versions"), filepath.Join(p.root, "versions")); err != nil {
			return retained
		}
	}
	if movedRecord {
		if _, err := os.Lstat(filepath.Join(p.root, "installed.json")); !os.IsNotExist(err) {
			return retained
		}
		if err := ops.rename(filepath.Join(stage, "installed.json"), filepath.Join(p.root, "installed.json")); err != nil {
			return retained
		}
	}
	if stage != "" {
		_ = os.Remove(stage)
	} // Empty directory only; never delete recovery payloads.
	if running {
		if err := ops.start(p.root, p.dshHome); err != nil {
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheUninstallDidNotFinishAndThePreviou))
		}
	}
	return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheUninstallDidNotFinishThePreviousVe))
}

func validateBridgeRemovalPaths(root string) error {
	// Refuse symlinked ancestors, payload roots and installation records. This
	// does not claim protection against privileged concurrent path replacement.
	abs, err := filepath.Abs(root)
	if err != nil {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsInvalidWhalebridgeComponentPath))
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeComponentDirectoryIsMissi))
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	for _, name := range []string{"versions", "data", "installed.json"} {
		info, err := os.Lstat(filepath.Join(root, name))
		if name == "data" && os.IsNotExist(err) {
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || name != "installed.json" && !info.IsDir() || name == "installed.json" && !info.Mode().IsRegular() {
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeProgramOrInstallRecordIs))
		}
	}
	return nil
}

func inspectBridgeRemoval(ctx context.Context, root, dshHome, version string) (bool, error) {
	var output []byte
	st, err := readBridgeState(filepath.Join(root, "data", "state.json"))
	if os.IsNotExist(err) {
		output, err = bridgeOfflineCommand(ctx, filepath.Join(root, "versions", version, bridgeAsset), filepath.Join(root, "data"), dshHome, "whalebridge-status")
	} else if err == nil {
		client := &http.Client{Timeout: 2 * time.Second}
		if err = verifyBridgeReady(ctx, st, version, st.PID, client); err != nil {
			return false, err
		}
		resp, requestErr := bridgeLocalRequest(ctx, st, http.MethodGet, "status", nil, client)
		if requestErr != nil {
			return false, requestErr
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, errors.New(storelocale.T(storelocale.KeyStoreComponentsCouldNotReadTheWhalebridgeUninstallImp2))
		}
		output, err = io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
	}
	if err != nil || len(output) > 64<<10 {
		return false, errors.New(storelocale.T(storelocale.KeyStoreComponentsCouldNotReadTheWhalebridgeUninstallImp))
	}
	var info struct {
		DefaultModel *bool  `json:"defaultModel"`
		PID          int    `json:"pid"`
		Version      string `json:"version"`
	}
	if json.Unmarshal(output, &info) != nil || info.DefaultModel == nil || st.PID != 0 && (info.PID != st.PID || info.Version != version) {
		return false, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeUninstallImpactStateIsIn))
	}
	return *info.DefaultModel, nil
}

type bridgeCommandOutput struct {
	bytes    []byte
	overflow bool
}

func (b *bridgeCommandOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := (64 << 10) - len(b.bytes)
	if n > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	b.bytes = append(b.bytes, p...)
	return n, nil
}

// These are existing official WhaleBridge commands, not shell interpolation or
// invented HTTP APIs. No credential value or command output enters an error.
func bridgeOfflineCommand(ctx context.Context, exe, data, dshHome, command string) ([]byte, error) {
	if !WhaleBridgeManagementSupported() || command != "whalebridge-status" && command != "whalebridge-disconnect" {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsUnsupportedWhalebridgeComponentCommand))
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, command)
	cmd.Dir = filepath.Dir(exe)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(name) {
		case "ELECTRON_RUN_AS_NODE", "MAGPIE_WEB_KEY", "MAGPIE_WEB_RUNKEY", "LAUNCHER_COMPONENT_ID", "LAUNCHER_COMPONENT_DATA_DIR", "WHALEBRIDGE_DSH_HOME", "MAGPIE_ADDR":
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "LAUNCHER_COMPONENT_ID=whalebridge", "LAUNCHER_COMPONENT_DATA_DIR="+data, "WHALEBRIDGE_DSH_HOME="+dshHome, "MAGPIE_ADDR=127.0.0.1:3427")
	var output bridgeCommandOutput
	cmd.Stdout, cmd.Stderr = &output, io.Discard
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil || output.overflow {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheWhalebridgeOfflineCommandDidNotFini))
	}
	return output.bytes, nil
}
