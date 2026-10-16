package components

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrUnknownAction means action is not install|start|stop|open.
var ErrUnknownAction = errors.New("components: unknown action")

// ErrNotInstalled means the component has no installed version.
var ErrNotInstalled = errors.New("components: not installed")

// RunAction performs a launcher component lifecycle action against userData.
// install fetches and verifies the official first-install release; start/stop/open
// work for whalebridge when a versions/<ver>/WhaleBridge-win32-x64.exe exists.
func RunAction(userData, id, action string) error {
	id = strings.TrimSpace(id)
	action = strings.TrimSpace(strings.ToLower(action))
	if id == "" || action == "" {
		return ErrUnknownAction
	}
	root := filepath.Join(userData, "components", id)
	switch action {
	case "install":
		if id != "whalebridge" {
			return fmt.Errorf("components: install unsupported for %s", id)
		}
		return InstallWhaleBridge(context.Background(), userData, "github")
	case "start":
		if id == "whalebridge" {
			return startWhalebridge(root, filepath.Join(userData, "dsh-home"))
		}
		return fmt.Errorf("components: start unsupported for %s", id)
	case "stop":
		if id == "whalebridge" {
			return stopWhalebridge(root)
		}
		return fmt.Errorf("components: stop unsupported for %s", id)
	case "open", "manage":
		if id == "whalebridge" {
			return openWhalebridge(root)
		}
		// Best-effort: open component data dir in Explorer / Finder.
		return openPath(filepath.Join(root, "data"))
	default:
		return ErrUnknownAction
	}
}

var bridgeStartMu sync.Mutex
var errBridgeExitUnconfirmed = errors.New("鲸桥启动失败，进程退出尚未确认，请查看组件日志")

func startWhalebridge(root, dshHome string) error {
	if !bridgeStartMu.TryLock() {
		return fmt.Errorf("鲸桥正在启动")
	}
	defer bridgeStartMu.Unlock()
	return startWhalebridgeLocked(root, dshHome)
}

func startWhalebridgeLocked(root, dshHome string) error {
	inst := readInstalled(filepath.Join(root, "installed.json"))
	ver := inst.Version
	if ver == "" {
		return ErrNotInstalled
	}
	if !bridgeVersion.MatchString(ver) {
		return fmt.Errorf("鲸桥安装版本无效")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	dataDir := filepath.Join(root, "data")
	statePath := filepath.Join(dataDir, "state.json")
	if st, err := readBridgeState(statePath); err == nil {
		// Never kill or replace an instance we did not start. A stale/foreign state
		// needs explicit lifecycle recovery, not an implicit duplicate launch.
		return verifyBridgeReady(ctx, st, ver, st.PID, client)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("鲸桥已有状态记录无法验证，请先检查组件日志")
	}
	exe := filepath.Join(root, "versions", ver, bridgeAsset)
	if info, err := os.Stat(exe); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("鲸桥程序缺失或不是常规文件")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	logF, err := os.OpenFile(filepath.Join(dataDir, "component.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer logF.Close()
	cmd := exec.Command(exe, "whalebridge")
	cmd.Dir = filepath.Dir(exe)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(name) {
		case "ELECTRON_RUN_AS_NODE", "MAGPIE_WEB_KEY", "MAGPIE_WEB_RUNKEY", "LAUNCHER_COMPONENT_ID", "LAUNCHER_COMPONENT_DATA_DIR", "WHALEBRIDGE_DSH_HOME", "MAGPIE_ADDR":
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "LAUNCHER_COMPONENT_ID=whalebridge", "LAUNCHER_COMPONENT_DATA_DIR="+dataDir, "WHALEBRIDGE_DSH_HOME="+dshHome, "MAGPIE_ADDR=127.0.0.1:3427")
	cmd.Stdout = logF
	cmd.Stderr = logF
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("鲸桥进程无法启动，请查看组件日志")
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait(); close(exited) }()
	if err := waitBridgeReady(ctx, statePath, ver, cmd.Process.Pid, exited, client); err != nil {
		// Only the process handle owned by this launch is eligible for cleanup.
		_ = cmd.Process.Kill()
		// Do not clear state or let rollback start a second instance until the
		// owned process has actually exited (a PID lookup is insufficient).
		select {
		case <-exited:
		case <-time.After(10 * time.Second):
			return errBridgeExitUnconfirmed
		}
		if st, e := readBridgeState(statePath); e == nil && st.PID == cmd.Process.Pid && st.Version == ver {
			_ = os.Remove(statePath)
		}
		return err
	}
	return nil
}

func stopWhalebridge(root string) error {
	if !bridgeStartMu.TryLock() {
		return fmt.Errorf("鲸桥正在启动或切换")
	}
	defer bridgeStartMu.Unlock()
	// A missing state remains an idempotent stop, including an uninstalled component.
	if _, err := os.Lstat(filepath.Join(root, "data", "state.json")); os.IsNotExist(err) {
		return nil
	}
	inst, err := checkedBridgeInstalled(root)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err = stopBridgeInstance(ctx, root, inst.Version, bridgeRuntime())
	return err
}

func openWhalebridge(root string) error {
	statePath := filepath.Join(root, "data", "state.json")
	b, err := os.ReadFile(statePath)
	if err != nil {
		return openPath(filepath.Join(root, "data"))
	}
	var st struct {
		URL   string `json:"url"`
		Ready bool   `json:"ready"`
	}
	if json.Unmarshal(b, &st) != nil || !st.Ready || strings.TrimSpace(st.URL) == "" {
		return openPath(filepath.Join(root, "data"))
	}
	u, err := bridgeManagementURL(st.URL)
	if err != nil {
		return err
	}
	return openURL(u.String())
}

// OpenInFileManager reveals path in Explorer / Finder / xdg-open.
func OpenInFileManager(path string) error {
	return openPath(path)
}

func openPath(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	_ = os.MkdirAll(path, 0o755)
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func openURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("empty url")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", raw).Start()
	case "darwin":
		return exec.Command("open", raw).Start()
	default:
		return exec.Command("xdg-open", raw).Start()
	}
}

// FormatActionError returns a short UI note for RunAction failures.
func FormatActionError(id, action string, err error) string {
	if err == nil {
		return id + " · " + action + " 完成"
	}
	if errors.Is(err, ErrNotInstalled) {
		return id + " 尚未安装"
	}
	if errors.Is(err, ErrUnknownAction) {
		return "未知操作: " + action
	}
	return id + " · " + action + ": " + err.Error()
}

// PIDString is a small helper for tests / diagnostics.
func PIDString(pid int) string {
	if pid <= 0 {
		return ""
	}
	return strconv.Itoa(pid)
}
