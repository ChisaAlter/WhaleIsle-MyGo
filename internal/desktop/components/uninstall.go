package components

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
		return nil, fmt.Errorf("鲸桥组件管理仅支持 Windows x64")
	}
	return prepareBridgeUninstall(ctx, userData, bridgeRemovalRuntime())
}

func prepareBridgeUninstall(ctx context.Context, userData string, ops bridgeRemovalOps) (*BridgeUninstall, error) {
	if !componentInstallMu.TryLock() {
		return nil, fmt.Errorf("组件安装或切换正在进行")
	}
	defer componentInstallMu.Unlock()
	if !bridgeStartMu.TryLock() {
		return nil, fmt.Errorf("鲸桥正在启动或停止")
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
		return "", fmt.Errorf("鲸桥组件管理仅支持 Windows x64")
	}
	return p.applyWithDataRemoval(ctx, bridgeRemovalRuntime(), removeData)
}

func (p *BridgeUninstall) apply(ctx context.Context, ops bridgeRemovalOps) (string, error) {
	return p.applyWithDataRemoval(ctx, ops, false)
}

func (p *BridgeUninstall) applyWithDataRemoval(ctx context.Context, ops bridgeRemovalOps, removeData bool) (string, error) {
	if !componentInstallMu.TryLock() {
		return "", fmt.Errorf("组件安装或切换正在进行")
	}
	defer componentInstallMu.Unlock()
	if !bridgeStartMu.TryLock() {
		return "", fmt.Errorf("鲸桥正在启动或停止")
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
		return "", fmt.Errorf("鲸桥安装版本已改变，请重新确认卸载目标")
	}
	if _, err := verifyBridgeVersion(p.root, p.Version()); err != nil {
		return "", err
	}
	impact, err := ops.inspect(ctx, p.root, p.dshHome, p.Version())
	if err != nil {
		return "", err
	}
	if impact != p.defaultModel {
		return "", fmt.Errorf("鲸桥默认模型状态已改变，请重新确认卸载影响")
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
		return "鲸桥已卸载并断开；程序文件清理未完成，已保留隔离文件。供应商配置、账户、用量和聊天记录已保留。", nil
	}
	if removeData {
		if err := ops.remove(p.DataDirectory()); err != nil {
			return "鲸桥已卸载并断开，但鲸桥配置清理未完成：" + err.Error() + "。聊天记录和其他渠道未删除。", nil
		}
	}
	return "", nil
}

// Recovery never overwrites a replacement owner or starts a duplicate process.
// On any uncertain recovery the quarantine is retained rather than destroyed.
func (p *BridgeUninstall) restoreRemoval(stage string, movedVersions, movedRecord, running bool, ops bridgeRemovalOps) error {
	retained := fmt.Errorf("卸载未完成，自动恢复未完成；程序和安装记录保留在组件目录的隔离文件夹，供应商数据未删除。请检查组件日志并恢复安装；同时检查默认模型和受管代理的接入状态")
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
			return fmt.Errorf("卸载未完成，已恢复原版本；原实例未就绪。请重试启动并检查默认模型及受管代理的接入状态")
		}
	}
	return fmt.Errorf("卸载未完成，已恢复原版本及原运行状态；供应商数据未删除。部分接入设置可能已断开，请检查默认模型和受管代理；程序恢复不代表这些接入设置已恢复")
}

func validateBridgeRemovalPaths(root string) error {
	// Refuse symlinked ancestors, payload roots and installation records. This
	// does not claim protection against privileged concurrent path replacement.
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("鲸桥组件路径无效")
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("鲸桥组件目录缺失或含链接，已保留")
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
			return fmt.Errorf("鲸桥程序或安装记录缺失或含链接，已保留")
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
			return false, fmt.Errorf("无法读取鲸桥卸载影响")
		}
		output, err = io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
	}
	if err != nil || len(output) > 64<<10 {
		return false, fmt.Errorf("无法读取鲸桥卸载影响，未停止组件")
	}
	var info struct {
		DefaultModel *bool  `json:"defaultModel"`
		PID          int    `json:"pid"`
		Version      string `json:"version"`
	}
	if json.Unmarshal(output, &info) != nil || info.DefaultModel == nil || st.PID != 0 && (info.PID != st.PID || info.Version != version) {
		return false, fmt.Errorf("鲸桥卸载影响状态无效，未停止组件")
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
		return nil, fmt.Errorf("鲸桥组件命令不受支持")
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
		return nil, fmt.Errorf("鲸桥离线命令未完成，请检查组件状态")
	}
	return output.bytes, nil
}
