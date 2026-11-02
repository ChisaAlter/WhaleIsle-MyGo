package installctrl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"whaleisle/internal/paths"
	"whaleisle/internal/processutil"
)

const (
	// WebProfile is the dsh profile used for desktop plugin installs.
	WebProfile = "web"
	// DefaultInstallTimeout bounds plugin add when no deadline is passed.
	DefaultInstallTimeout = 10 * time.Minute
	maxInstallLogBytes    = 256 * 1024
)

// CommandRunner runs an install subprocess (overridden in tests).
type CommandRunner func(ctx context.Context, dir, name string, args ...string) (exitCode int, log string, err error)

// InstallConfig locates the web profile and toolchain for plugin add.
type InstallConfig struct {
	ProfileDir          string
	DshHome             string
	HarnessRoot         string
	NodeBin             string
	Timeout             time.Duration
	Runner              CommandRunner
	HomeDir             string
	ExpectedPackageName string // confirmed manifest name, including GitHub aliases
}

// InstallReady reports whether node and the vendored dsh CLI entry exist.
func InstallReady(cfg InstallConfig) bool {
	if strings.TrimSpace(cfg.NodeBin) == "" {
		return false
	}
	if _, err := os.Stat(cfg.NodeBin); err != nil {
		return false
	}
	bin := harnessBinJS(cfg.HarnessRoot)
	if bin == "" {
		return false
	}
	if _, err := os.Stat(bin); err != nil {
		return false
	}
	return true
}

func harnessBinJS(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "apps", "cli", "lib", "bin.js")
}

// PNPMInstall returns an InstallFunc that runs `dsh plugin --profile web add`
// in the web profile (pnpm under the hood), matching the Electron desktop flow.
func PNPMInstall(cfg InstallConfig) InstallFunc {
	return PNPMInstallContext(context.Background(), cfg)
}

// PNPMInstallContext shares the marketplace installer while allowing import
// cancellation to terminate the current CLI command and restore its snapshot.
func PNPMInstallContext(ctx context.Context, cfg InstallConfig) InstallFunc {
	return func(spec string, opts InstallOpts) Result {
		return runPluginInstallContext(ctx, cfg, spec, opts)
	}
}

func runPluginInstall(cfg InstallConfig, spec string, opts InstallOpts) Result {
	return runPluginInstallContext(context.Background(), cfg, spec, opts)
}

func runPluginInstallContext(parent context.Context, cfg InstallConfig, spec string, opts InstallOpts) Result {
	spec = strings.TrimSpace(spec)
	out := Result{Spec: spec, AllowBuilds: []string{}}
	if err := parent.Err(); err != nil {
		out.Error = err.Error()
		return out
	}

	if cfg.ExpectedPackageName != "" && RegistryPluginName(cfg.ExpectedPackageName+"@0.0.0") != cfg.ExpectedPackageName {
		out.Error = "预期插件包名无效"
		return out
	}

	if !InstallReady(cfg) {
		out.Error = "未找到 Node.js 或 dsh CLI；请安装 Node.js 22+ 并构建 vendored harness"
		return out
	}

	profile := strings.TrimSpace(cfg.ProfileDir)
	if profile == "" {
		out.Error = "web profile 路径未配置"
		return out
	}
	if err := os.MkdirAll(profile, 0o755); err != nil {
		out.Error = fmt.Sprintf("无法创建 profile 目录：%v", err)
		return out
	}

	allowBuilds := NormalizeAllowBuilds(opts.AllowBuilds)
	if allowBuilds == nil {
		out.Error = "allowBuilds 包含非法包名"
		return out
	}
	beforeModules := ListNodeModuleNames(profile)
	snap, snapErr := CaptureInstallSnapshot(profile)
	if snapErr != nil {
		out.Error = "无法创建安装回滚点，未执行安装：" + snapErr.Error()
		return out
	}

	// Capture approval-file changes too, so cancellation/failure never leaves
	// an approved-build mutation outside the same rollback boundary.
	if len(allowBuilds) > 0 {
		if _, err := mergeAllowBuildsIntoWorkspace(profile, allowBuilds); err != nil {
			out.Error = err.Error()
			return RollbackInstall(snap, out)
		}
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultInstallTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	runner := cfg.Runner
	if runner == nil {
		runner = execPluginRunner(cfg)
	}
	home := cfg.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}

	bin := harnessBinJS(cfg.HarnessRoot)
	args := []string{bin, "plugin", "--profile", WebProfile, "add", spec}
	code, log, runErr := runner(ctx, home, cfg.NodeBin, args...)
	log = truncateLog(log)
	out.Log = log

	fail := func(msg string) Result {
		out.OK = false
		out.Error = msg
		return RollbackInstall(snap, out)
	}

	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fail("安装超时")
		}
		return fail("安装已取消")
	}
	if runErr != nil && !errors.Is(runErr, errExitCode) {
		return fail(runErr.Error())
	}

	parsed := ParseAllowBuilds(log)
	needs := IsBuildApprovalFailure(code, parsed)
	out.NeedsAllowBuilds = needs
	out.AllowBuilds = parsed
	if code != 0 {
		if needs {
			return fail("需要允许该插件在本机执行构建脚本")
		}
		return fail(classifyInstallFailure(log))
	}

	afterModules := ListNodeModuleNames(profile)
	names := ResolveAddedPackageNames(spec, beforeModules, afterModules)
	if cfg.ExpectedPackageName != "" {
		// Reinstalls do not add a directory, and pnpm packages may be symlinks.
		// The confirmed manifest name is authoritative; a GitHub repo basename
		// is only a fallback for manual installs without an expected identity.
		names = []string{cfg.ExpectedPackageName}
	}
	if msg := ValidateLoadablePackages(profile, names); msg != "" {
		out.NeedsAllowBuilds = false
		out.AllowBuilds = []string{}
		return fail(msg)
	}
	out.OK = true
	return DiscardInstallSnapshot(snap, out)
}

var errExitCode = errors.New("non-zero exit")

func execPluginRunner(cfg InstallConfig) CommandRunner {
	return func(ctx context.Context, dir, name string, args ...string) (int, string, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		processutil.BackgroundTree(cmd)
		cmd.Dir = dir
		cmd.Env = pluginChildEnv(cfg)
		combined, err := cmd.CombinedOutput()
		log := string(combined)
		if err == nil {
			return 0, log, nil
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code := exit.ExitCode()
			return code, log, errExitCode
		}
		return 1, log, err
	}
}

func pluginChildEnv(cfg InstallConfig) []string {
	env := make([]string, 0, len(os.Environ())+6)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(key) {
		case "DSH_HOME", "DSH_HARNESS_ROOT", "ELECTRON_RUN_AS_NODE", "ELECTRON_NO_ASAR",
			"DSH_DESKTOP_INSTALL_URL", "DSH_DESKTOP_INSTALL_TOKEN":
			continue
		}
		env = append(env, kv)
	}
	if home := strings.TrimSpace(cfg.DshHome); home != "" {
		env = append(env, "DSH_HOME="+home)
	}
	if root := strings.TrimSpace(cfg.HarnessRoot); root != "" {
		env = append(env, "DSH_HARNESS_ROOT="+root)
	}
	env = append(env, "CI=1", "npm_config_yes=true", "npm_config_update_notifier=false")
	if extra := prependPnpmToPath(cfg); extra != "" {
		env = append(env, "PATH="+extra+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	return env
}

func prependPnpmToPath(cfg InstallConfig) string {
	if bin := findPnpmBin(); bin != "" {
		return filepath.Dir(bin)
	}
	if cjs := findPnpmCJS(cfg.HarnessRoot); cjs != "" && cfg.NodeBin != "" {
		shimDir, err := ensurePnpmShimDir(cfg.NodeBin, cjs)
		if err == nil && shimDir != "" {
			return shimDir
		}
	}
	return ""
}

func findPnpmBin() string {
	// LookPath applies PATHEXT within each Windows PATH directory. Searching
	// every directory for .cmd first would let a later global shim override an
	// earlier package-local pnpm.exe, defeating the selected child toolchain.
	if bin, err := exec.LookPath("pnpm"); err == nil {
		return bin
	}
	return ""
}

func findPnpmCJS(harnessRoot string) string {
	root := strings.TrimSpace(harnessRoot)
	candidates := []string{}
	if root != "" {
		candidates = append(candidates,
			filepath.Join(root, "node_modules", "pnpm", "bin", "pnpm.cjs"),
		)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func ensurePnpmShimDir(nodeBin, pnpmCJS string) (string, error) {
	dir := filepath.Join(paths.UserDataDir(), "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		cmdPath := filepath.Join(dir, "pnpm.cmd")
		body := fmt.Sprintf("@echo off\r\n\"%s\" \"%s\" %%*\r\n", nodeBin, pnpmCJS)
		if err := os.WriteFile(cmdPath, []byte(body), 0o644); err != nil {
			return "", err
		}
	} else {
		shPath := filepath.Join(dir, "pnpm")
		body := fmt.Sprintf("#!/bin/sh\nexec \"%s\" \"%s\" \"$@\"\n", nodeBin, pnpmCJS)
		if err := os.WriteFile(shPath, []byte(body), 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

var allowBuildsSectionRE = regexp.MustCompile(`(?m)^allowBuilds\s*:`)

func mergeAllowBuildsIntoWorkspace(profileDir string, keys []string) (string, error) {
	file := filepath.Join(profileDir, "pnpm-workspace.yaml")
	text := ""
	if data, err := os.ReadFile(file); err == nil {
		text = string(data)
	}
	if !allowBuildsSectionRE.MatchString(text) {
		text = strings.TrimRight(text, " \t\r\n") + "\n"
		if text == "\n" {
			text = ""
		}
		if text != "" {
			text += "\n"
		}
		text += "allowBuilds:\n"
	}
	for _, key := range keys {
		quoted := fmt.Sprintf("%q", key)
		pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(quoted) + `\s*:`)
		if pattern.MatchString(text) {
			continue
		}
		text = allowBuildsSectionRE.ReplaceAllString(text, "allowBuilds:\n  "+quoted+": true\n")
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", fmt.Errorf("无法写入 pnpm-workspace.yaml：%w", err)
	}
	tmp := file + ".tmp"
	payload := text
	if !strings.HasSuffix(payload, "\n") {
		payload += "\n"
	}
	if err := os.WriteFile(tmp, []byte(payload), 0o644); err != nil {
		return "", fmt.Errorf("无法写入 pnpm-workspace.yaml：%w", err)
	}
	if err := os.Rename(tmp, file); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("无法写入 pnpm-workspace.yaml：%w", err)
	}
	return file, nil
}

func classifyInstallFailure(log string) string {
	switch {
	case strings.Contains(log, "ERR_PNPM_UNEXPECTED_STORE"),
		strings.Contains(log, "ERR_PNPM_UNEXPECTED_VIRTUAL_STORE"),
		strings.Contains(log, "ERR_PNPM_MODULES_BREAKING_CHANGE"):
		return "依赖目录与当前 pnpm 版本不兼容；请保留日志后修复 profile"
	case strings.Contains(log, "ERR_PNPM_NO_MATCHING_VERSION"),
		strings.Contains(log, "ERR_PNPM_NO_MATCHING_VERSION_INSIDE_WORKSPACE"):
		return "目标包版本不可用；请稍后重试"
	case strings.Contains(log, "ETIMEDOUT"),
		strings.Contains(log, "ECONNRESET"),
		strings.Contains(log, "ENOTFOUND"),
		strings.Contains(log, "EAI_AGAIN"),
		strings.Contains(log, "ECONNREFUSED"),
		strings.Contains(strings.ToLower(log), "fetch failed"):
		return "下载失败：请检查网络、代理及仓库访问权限后重试"
	default:
		return "安装失败"
	}
}

func truncateLog(log string) string {
	if len(log) <= maxInstallLogBytes {
		return log
	}
	return log[:maxInstallLogBytes] + "\n…(log truncated)"
}
