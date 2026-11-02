package importer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/processutil"
)

// TestNSISInstallerSmoke runs scripts/smoke-nsis.ps1 when makensis is available.
func TestNSISInstallerSmoke(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("NSIS smoke is Windows-only")
	}
	root := findRepoRoot(t)
	if resolveMakensis(root) == "" {
		t.Skip("makensis not on PATH and .toolchain/nsis/makensis.exe missing")
	}

	ps1 := filepath.Join(root, "scripts", "smoke-nsis.ps1")
	if _, err := os.Stat(ps1); err != nil {
		t.Fatalf("smoke script: %v", err)
	}

	output := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", ps1, "-OutputRoot", output)
	processutil.Background(cmd)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "WHALEISLE_PACKAGE_SKIP_RUNTIME=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("smoke-nsis.ps1: %v\n%s", err, out)
	}

	setups, err := filepath.Glob(filepath.Join(output, "WhaleIsle-Smoke-*.exe"))
	if err != nil || len(setups) != 1 {
		t.Fatalf("expected one fresh installer, got %v: %v", setups, err)
	}
	info, err := os.Stat(setups[0])
	if err != nil {
		t.Fatalf("installer: %v", err)
	}
	if info.Size() <= 0 {
		t.Fatalf("installer size %d", info.Size())
	}
}

// TestNSISSilentInstallSmoke compiles then silently installs into TEMP and uninstalls.
// Never writes to %LOCALAPPDATA%\Programs\Deepseek-Harness-Desktop.
func TestNSISSilentInstallSmoke(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("NSIS smoke is Windows-only")
	}
	root := findRepoRoot(t)
	if resolveMakensis(root) == "" {
		t.Skip("makensis not on PATH and .toolchain/nsis/makensis.exe missing")
	}

	ps1 := filepath.Join(root, "scripts", "smoke-nsis.ps1")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", ps1, "-SilentInstall", "-OutputRoot", t.TempDir())
	processutil.Background(cmd)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "WHALEISLE_PACKAGE_SKIP_RUNTIME=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("smoke-nsis.ps1 -SilentInstall: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Silent install/uninstall smoke: passed") {
		t.Fatalf("missing silent-install success marker\n%s", out)
	}
}

func resolveMakensis(repoRoot string) string {
	if p, err := exec.LookPath("makensis"); err == nil {
		return p
	}
	local := filepath.Join(repoRoot, ".toolchain", "nsis", "makensis.exe")
	if st, err := os.Stat(local); err == nil && !st.IsDir() {
		return local
	}
	return ""
}
