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
		return nil, fmt.Errorf("鲸桥仅支持 Windows x64")
	}
	endpoint, err := bridgeReleaseEndpoint(route)
	if err != nil {
		return nil, err
	}
	return prepareBridgeUpdate(ctx, userData, endpoint, &http.Client{Timeout: 5 * time.Minute}, false)
}

func prepareBridgeUpdate(ctx context.Context, userData, endpoint string, client *http.Client, fixture bool) (*BridgeChange, error) {
	if !componentInstallMu.TryLock() {
		return nil, fmt.Errorf("组件安装或切换正在进行")
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
		return nil, fmt.Errorf("当前线路没有较新版本；降级请使用回滚")
	}
	if _, err := stageBridgeRelease(ctx, p.root, m, asset, client, fixture); err != nil {
		return nil, err
	}
	p.version = m.Version
	return p, nil
}

func PrepareWhaleBridgeRollback(userData string) (*BridgeChange, error) {
	if !componentInstallMu.TryLock() {
		return nil, fmt.Errorf("组件安装或切换正在进行")
	}
	defer componentInstallMu.Unlock()
	p, err := newBridgeChange(userData)
	if err != nil {
		return nil, err
	}
	if !bridgeVersion.MatchString(p.from.Previous) || p.from.Previous == p.from.Version {
		return nil, fmt.Errorf("没有可回滚版本")
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
		return inst, fmt.Errorf("鲸桥安装记录无效，已保留现有文件")
	}
	return inst, nil
}

// verifyBridgeVersion also protects rollback and cached staged downloads: neither
// a symlink nor a changed executable may be promoted using only a version name.
func verifyBridgeVersion(root, version string) (bridgeReleaseManifest, error) {
	var m bridgeReleaseManifest
	if !bridgeVersion.MatchString(version) {
		return m, fmt.Errorf("鲸桥版本无效")
	}
	dir := filepath.Join(root, "versions", version)
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return m, fmt.Errorf("鲸桥版本目录缺失或无效")
	}
	for _, name := range []string{bridgeManifest, bridgeAsset} {
		if info, err := os.Lstat(filepath.Join(dir, name)); err != nil || !info.Mode().IsRegular() {
			return m, fmt.Errorf("鲸桥版本文件缺失或不是常规文件")
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, bridgeManifest))
	if err != nil {
		return m, err
	}
	if json.Unmarshal(b, &m) != nil || validateBridgeManifest(m) != nil || m.Version != version {
		return m, fmt.Errorf("鲸桥本地版本清单无效")
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
		return m, fmt.Errorf("鲸桥本地程序大小或 SHA256 校验失败")
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
		return fmt.Errorf("组件安装或切换正在进行")
	}
	defer componentInstallMu.Unlock()
	if !bridgeStartMu.TryLock() {
		return fmt.Errorf("鲸桥正在启动或停止")
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
		return fmt.Errorf("鲸桥版本已改变，请重新检查更新或回滚")
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
		return fmt.Errorf("版本切换未完成，实例退出无法确认；已保留全部版本和数据：%w", cause)
	}
	if err := replaceBridgeRecord(p.root, old); err != nil {
		return fmt.Errorf("版本切换失败且无法恢复安装记录，旧版本和数据仍保留：%w", err)
	}
	if running {
		if err := life.start(p.root, p.dshHome); err != nil {
			return fmt.Errorf("版本切换失败，已恢复旧版本记录，但旧实例未就绪：%w", err)
		}
	}
	return fmt.Errorf("版本切换未生效，已恢复原版本：%w", cause)
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
		return false, fmt.Errorf("鲸桥现有实例身份不一致，已保留")
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
		return false, fmt.Errorf("鲸桥实例已改变，已取消切换")
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
		return true, fmt.Errorf("鲸桥停止后状态已改变，已保留新状态")
	}
	return true, nil
}
