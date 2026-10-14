package harness

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// npmPackage is the published dsh CLI (legacy npx fallback).
const npmPackage = "@deepseek-ai/dsh"

// portScanRange is how many ports after the configured one are tried.
const portScanRange = 50

// LaunchOptions select the dsh launcher and how it runs.
type LaunchOptions struct {
	// DshBin is a configured launcher: a bin.js run with Node, or an executable.
	// When set, it wins over the vendored harness (used by tests).
	DshBin string
	// NodeBin is a configured Node.js; empty finds one on PATH / Program Files.
	NodeBin string
	// Version is the npm version npx runs when no launcher is installed.
	Version string
	// Host and Port are where dsh web listens; Port is the first one tried.
	Host string
	Port int
	// Home is the DSH_HOME the desktop keeps apart from the CLI's ~/.dsh.
	Home string
	// HarnessRoot is the vendored deepseek-harness tree. Empty skips vendor
	// unless VendorDir is set and contains deepseek-harness.
	HarnessRoot string
	// VendorDir holds deepseek-harness and desktop plugins. Used with
	// Prepare; also locates HarnessRoot when HarnessRoot is empty.
	VendorDir string
	// PatchFiles are absolute paths passed as repeated --patch flags.
	PatchFiles []string
	// SkipUserPlugins starts dsh in plugin recovery mode.
	SkipUserPlugins bool
	// AllowNpx enables the legacy npx fallback when vendor/PATH fail.
	AllowNpx bool
	// ExtraEnv is appended to the child environment (install control URL/token, etc.).
	ExtraEnv []string
}

// Plan is one resolved command line.
type Plan struct {
	Command string
	Args    []string
	Env     []string
	// Source names where the launcher came from, for the boot log.
	Source string
	Host   string
	Port   int
	// Kind is "source", "configured", "PATH", or "npx".
	Kind string
}

// lookPath is exec.LookPath, replaced in tests.
var lookPath = exec.LookPath

// Resolve chooses the launcher: configured DshBin, then vendored harness
// (node + bin.js), then dsh on PATH, then optionally npx.
func Resolve(o LaunchOptions, port int) (Plan, error) {
	web := []string{"web"}
	if o.SkipUserPlugins {
		web = append(web, "--skip-user-plugins")
	}
	for _, p := range o.PatchFiles {
		if p != "" {
			web = append(web, "--patch", p)
		}
	}
	web = append(web, "--host", o.Host, "--port", strconv.Itoa(port), "--no-open")
	plan := Plan{Env: childEnv(o), Host: o.Host, Port: port}

	if o.DshBin != "" {
		if _, err := os.Stat(o.DshBin); err != nil {
			return Plan{}, fmt.Errorf("configured dshBin %s: %w", o.DshBin, err)
		}
		if isScript(o.DshBin) {
			// A configured path to the same source CLI must not bypass current-host
			// safety validation merely because it precedes vendor resolution.
			clean := filepath.Clean(o.DshBin)
			if strings.HasSuffix(clean, filepath.Join("apps", "cli", "lib", "bin.js")) {
				root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(clean))))
				if err := ValidateCurrentHostForLaunch(root); err != nil {
					return Plan{}, err
				}
			}
			node, err := resolveNode(o.NodeBin)
			if err != nil {
				return Plan{}, err
			}
			plan.Command, plan.Args = node, append([]string{o.DshBin}, web...)
		} else {
			plan.Command, plan.Args = o.DshBin, web
		}
		plan.Source = "configured " + o.DshBin
		plan.Kind = "configured"
		return plan, nil
	}

	if root := resolveHarnessRoot(o); root != "" {
		if err := ValidateCurrentHostForLaunch(root); err != nil {
			return Plan{}, err
		}
		binJS := filepath.Join(root, "apps", "cli", "lib", "bin.js")
		if _, err := os.Stat(binJS); err != nil {
			return Plan{}, fmt.Errorf("vendored harness missing bin.js at %s: %w", binJS, err)
		}
		webDist := filepath.Join(root, "apps", "web", "dist", "index.html")
		if _, err := os.Stat(webDist); err != nil {
			return Plan{}, fmt.Errorf("vendored harness not built (missing %s); run scripts/build-harness.ps1", webDist)
		}
		node, err := resolveNode(o.NodeBin)
		if err != nil {
			return Plan{}, err
		}
		plan.Command, plan.Args = node, append([]string{binJS}, web...)
		plan.Source = "vendor " + binJS
		plan.Kind = "source"
		return plan, nil
	}

	if bin, err := lookPath("dsh"); err == nil {
		plan.Command, plan.Args, plan.Source = bin, web, "PATH "+bin
		plan.Kind = "PATH"
		return plan, nil
	}

	if !o.AllowNpx {
		return Plan{}, errors.New("vendored harness not found; set WHALEISLE_VENDOR or run scripts/build-harness.ps1")
	}

	node, err := resolveNode(o.NodeBin)
	if err != nil {
		return Plan{}, err
	}
	npx, err := resolveNpx(node)
	if err != nil {
		return Plan{}, err
	}
	version := o.Version
	if version == "" {
		version = "latest"
	}
	spec := npmPackage + "@" + version
	plan.Command, plan.Args = npx, append([]string{"--yes", spec}, web...)
	plan.Source = "npx " + spec
	plan.Kind = "npx"
	return plan, nil
}

func resolveHarnessRoot(o LaunchOptions) string {
	if o.HarnessRoot != "" {
		if st, err := os.Stat(o.HarnessRoot); err == nil && st.IsDir() {
			return o.HarnessRoot
		}
		return ""
	}
	if o.VendorDir != "" {
		root := filepath.Join(o.VendorDir, "deepseek-harness")
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			return root
		}
	}
	return ""
}

func isScript(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".mjs", ".cjs":
		return true
	}
	return false
}

func resolveNode(configured string) (string, error) {
	if configured != "" {
		if _, err := os.Stat(configured); err != nil {
			return "", fmt.Errorf("configured nodeBin %s: %w", configured, err)
		}
		if strings.Contains(strings.ToLower(configured), "electron") {
			return "", fmt.Errorf("configured nodeBin looks like Electron: %s", configured)
		}
		return configured, nil
	}
	if bin, err := lookPath("node"); err == nil {
		if !strings.Contains(strings.ToLower(bin), "electron") {
			return bin, nil
		}
	}
	if runtime.GOOS == "windows" {
		for _, dir := range []string{
			`C:\AI\tools\node-v24.21.0-win-x64`,
			`C:\Program Files\nodejs`,
			`C:\Program Files (x86)\nodejs`,
		} {
			candidate := filepath.Join(dir, "node.exe")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}
	return "", errors.New("Node.js 22.19+ is required to run DeepSeek Harness; install it or set nodeBin")
}

func resolveNpx(node string) (string, error) {
	dir := filepath.Dir(node)
	names := []string{"npx"}
	if runtime.GOOS == "windows" {
		names = []string{"npx.cmd", "npx.exe"}
	}
	for _, name := range names {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	for _, name := range names {
		if bin, err := lookPath(name); err == nil {
			return bin, nil
		}
	}
	return "", fmt.Errorf("npx was not found next to %s; install dsh with `npm i -g %s` or set dshBin", node, npmPackage)
}

// childEnv is the parent environment with DSH_HOME / DSH_HARNESS_ROOT set
// and Electron's node switch removed.
func childEnv(o LaunchOptions) []string {
	env := make([]string, 0, len(os.Environ())+4)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(key) {
		case "DSH_HOME", "DSH_HARNESS_ROOT", "ELECTRON_RUN_AS_NODE", "ELECTRON_NO_ASAR",
			"DSH_DESKTOP_INSTALL_URL", "DSH_DESKTOP_INSTALL_TOKEN",
			"DSHD_TASK_CONTROL_TOKEN", "DSHD_DSHBOT_ENABLED", "DSHD_SCHEDULE_ENABLED":
			continue
		}
		env = append(env, kv)
	}
	if o.Home != "" {
		env = append(env, "DSH_HOME="+o.Home)
	}
	if root := resolveHarnessRoot(o); root != "" {
		env = append(env, "DSH_HARNESS_ROOT="+root)
	}
	env = append(env, "npm_config_yes=true", "npm_config_update_notifier=false")
	env = append(env, o.ExtraEnv...)
	return env
}

// FreePort returns the first port from start on host that accepts a listener.
func FreePort(host string, start int) (int, error) {
	for port := start; port < start+portScanRange && port <= 65535; port++ {
		l, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err == nil {
			l.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port in %d-%d on %s", start, start+portScanRange-1, host)
}

// readyLine matches the line dsh web prints once it serves.
var readyLine = regexp.MustCompile(`dsh web:\s*(https?://\S+)`)

// ParseReady returns the URL in a `dsh web: <url>` line.
func ParseReady(line string) (string, bool) {
	m := readyLine.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[1], true
}
