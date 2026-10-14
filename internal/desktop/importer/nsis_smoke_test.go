package importer

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", ps1)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "WHALEISLE_PACKAGE_SKIP_RUNTIME=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("smoke-nsis.ps1: %v\n%s", err, out)
	}

	setup := filepath.Join(root, "dist", "WhaleIsle-Setup-0.0.0-dev.exe")
	info, err := os.Stat(setup)
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
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", ps1, "-SilentInstall")
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
