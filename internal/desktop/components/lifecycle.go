package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ErrUnknownAction means action is not install|start|stop|open.
var ErrUnknownAction = errors.New("components: unknown action")

// ErrNotInstalled means the component has no installed version.
var ErrNotInstalled = errors.New("components: not installed")

// RunAction performs a launcher component lifecycle action against userData.
// install is a no-op stub until release download is ported; start/stop/open
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
		return fmt.Errorf("components: install not wired (use Electron release catalog)")
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

func startWhalebridge(root, dshHome string) error {
	inst := readInstalled(filepath.Join(root, "installed.json"))
	ver := inst.Version
	if ver == "" {
		ver = newestVersionDir(filepath.Join(root, "versions"))
	}
	if ver == "" {
		return ErrNotInstalled
	}
	exe := filepath.Join(root, "versions", ver, "WhaleBridge-win32-x64.exe")
	if runtime.GOOS != "windows" {
		// Dev / non-Windows: look for any executable under versions/ver.
		if _, err := os.Stat(exe); err != nil {
			alt := filepath.Join(root, "versions", ver)
			entries, _ := os.ReadDir(alt)
			for _, e := range entries {
				if !e.IsDir() {
					exe = filepath.Join(alt, e.Name())
					break
				}
			}
		}
	}
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("whalebridge binary missing: %w", err)
	}
	if readStatePID(filepath.Join(root, "data", "state.json")) {
		return nil // already running
	}
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(dataDir, "state.json"))
	logPath := filepath.Join(dataDir, "component.log")
	logF, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logF.Close()

	cmd := exec.Command(exe, "whalebridge")
	cmd.Dir = filepath.Dir(exe)
	cmd.Env = append(os.Environ(),
		"LAUNCHER_COMPONENT_ID=whalebridge",
		"LAUNCHER_COMPONENT_DATA_DIR="+dataDir,
		"WHALEBRIDGE_DSH_HOME="+dshHome,
		"MAGPIE_ADDR=127.0.0.1:3427",
	)
	cmd.Stdout = logF
	cmd.Stderr = logF
	if err := cmd.Start(); err != nil {
		return err
	}
	// Wait briefly for ready state.json (Electron polls up to 30s).
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if readStatePID(filepath.Join(dataDir, "state.json")) {
			_ = cmd.Process.Release()
			return nil
		}
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return fmt.Errorf("whalebridge exited early; see %s", logPath)
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Leave process running; UI will pick up state when ready.
	_ = cmd.Process.Release()
	return nil
}

func stopWhalebridge(root string) error {
	statePath := filepath.Join(root, "data", "state.json")
	b, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var st struct {
		PID int `json:"pid"`
	}
	if json.Unmarshal(b, &st) != nil || st.PID <= 0 {
		_ = os.Remove(statePath)
		return nil
	}
	proc, err := os.FindProcess(st.PID)
	if err == nil && proc != nil {
		_ = proc.Kill()
	}
	_ = os.Remove(statePath)
	return nil
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
	return openURL(st.URL)
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
