package market

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/paths"
)

// InstallOptions optionally overrides Node.js location (config nodeBin).
type InstallOptions struct {
	NodeBin string
}

// InstallGithubSpec runs dsh plugin add for github:owner/repo[#ref] in the web
// profile, matching the loopback installctrl server.
func InstallGithubSpec(spec string, allowBuilds []string, opts InstallOptions) installctrl.Result {
	spec = strings.TrimSpace(spec)
	if !installctrl.IsValidGithubSpec(spec) {
		return installctrl.Result{
			OK:    false,
			Spec:  spec,
			Error: "仅支持 github:owner/repo[#ref] 安装规格",
		}
	}
	normalized := installctrl.NormalizeAllowBuilds(allowBuilds)
	if normalized == nil {
		return installctrl.Result{
			OK:    false,
			Spec:  spec,
			Error: "allowBuilds 包含非法包名",
		}
	}

	cfg := installctrl.InstallConfig{
		ProfileDir:  paths.WebProfileDir(),
		DshHome:     paths.DshHome(),
		HarnessRoot: paths.HarnessRoot(),
	}
	if node, err := resolveNode(opts.NodeBin); err == nil {
		cfg.NodeBin = node
	}

	if !installctrl.InstallReady(cfg) {
		msg := "desktop plugin install requires Node.js and a built vendored harness (setup-harness)"
		if _, err := resolveNode(opts.NodeBin); err != nil {
			msg = err.Error()
		}
		return installctrl.Result{OK: false, Spec: spec, Error: msg}
	}

	return installctrl.PNPMInstall(cfg)(spec, installctrl.InstallOpts{AllowBuilds: normalized})
}

// InstallReady reports whether github:spec installs can run on this machine.
func InstallReady(nodeBin string) bool {
	cfg := installctrl.InstallConfig{
		ProfileDir:  paths.WebProfileDir(),
		DshHome:     paths.DshHome(),
		HarnessRoot: paths.HarnessRoot(),
	}
	if node, err := resolveNode(nodeBin); err == nil {
		cfg.NodeBin = node
	}
	return installctrl.InstallReady(cfg)
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
	if bin, err := exec.LookPath("node"); err == nil {
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
