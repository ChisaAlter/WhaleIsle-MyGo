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

// npmPackage is the published dsh CLI.
const npmPackage = "@deepseek-ai/dsh"

// portScanRange is how many ports after the configured one are tried.
const portScanRange = 50

// LaunchOptions select the dsh launcher and how it runs.
type LaunchOptions struct {
	// DshBin is a configured launcher: a bin.js run with Node, or an executable.
	DshBin string
	// NodeBin is a configured Node.js; empty finds one on PATH.
	NodeBin string
	// Version is the npm version npx runs when no launcher is installed.
	Version string
	// Host and Port are where dsh web listens; Port is the first one tried.
	Host string
	Port int
	// Home is the DSH_HOME the desktop keeps apart from the CLI's ~/.dsh.
	Home string
	// SkipUserPlugins starts dsh in plugin recovery mode.
	SkipUserPlugins bool
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
}

// lookPath is exec.LookPath, replaced in tests.
var lookPath = exec.LookPath

// Resolve chooses the launcher: the configured one, then dsh on PATH, then
// npx with the pinned npm version.
func Resolve(o LaunchOptions, port int) (Plan, error) {
	web := []string{"web", "--host", o.Host, "--port", strconv.Itoa(port), "--no-open"}
	if o.SkipUserPlugins {
		web = append(web, "--skip-user-plugins")
	}
	plan := Plan{Env: childEnv(o.Home), Host: o.Host, Port: port}

	if o.DshBin != "" {
		if _, err := os.Stat(o.DshBin); err != nil {
			return Plan{}, fmt.Errorf("configured dshBin %s: %w", o.DshBin, err)
		}
		if isScript(o.DshBin) {
			node, err := resolveNode(o.NodeBin)
			if err != nil {
				return Plan{}, err
			}
			plan.Command, plan.Args = node, append([]string{o.DshBin}, web...)
		} else {
			plan.Command, plan.Args = o.DshBin, web
		}
		plan.Source = "configured " + o.DshBin
		return plan, nil
	}
	if bin, err := lookPath("dsh"); err == nil {
		plan.Command, plan.Args, plan.Source = bin, web, "PATH "+bin
		return plan, nil
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
	return plan, nil
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
		return configured, nil
	}
	if bin, err := lookPath("node"); err == nil {
		return bin, nil
	}
	if runtime.GOOS == "windows" {
		for _, dir := range []string{`C:\Program Files\nodejs`, `C:\Program Files (x86)\nodejs`} {
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

// childEnv is the parent environment with DSH_HOME set to the desktop home
// and Electron's node switch removed.
func childEnv(home string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(key) {
		case "DSH_HOME", "ELECTRON_RUN_AS_NODE":
			continue
		}
		env = append(env, kv)
	}
	if home != "" {
		env = append(env, "DSH_HOME="+home)
	}
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
