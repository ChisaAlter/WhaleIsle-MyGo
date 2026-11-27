package mygo_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Link the real product entry point; an empty example can shed its entire UI.
func TestReleaseBuildStripsTestHooks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping product build in short mode")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release-build.json"))
	if os.IsNotExist(err) {
		t.Skip("product release contract absent in standalone MyGo checkout")
	}
	if err != nil {
		t.Fatal(err)
	}
	var contract struct{ ForbiddenSymbols []string }
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "whaleisle-release-symbols.exe")
	shell := "pwsh"
	if _, err := exec.LookPath(shell); err != nil && runtime.GOOS == "windows" {
		shell = "powershell"
	}
	build := exec.Command(shell, "-NoProfile", "-File", filepath.Join(root, "scripts", "build-release.ps1"), "-Package", "cmd/whaleisle", "-Output", bin, "-KeepSymbols")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("real release build: %v\n%s", err, out)
	}
	nm := exec.Command("go", "tool", "nm", bin)
	nm.Dir = root
	out, err := nm.CombinedOutput()
	if err != nil {
		t.Fatalf("release symbols: %v\n%s", err, out)
	}
	symbols := string(out)
	if !strings.Contains(symbols, "main.main") || !strings.Contains(symbols, "github.com/egoist/mygo") {
		t.Fatal("product window implementation was not linked")
	}
	for _, forbidden := range contract.ForbiddenSymbols {
		for _, line := range strings.Split(symbols, "\n") {
			if strings.Contains(line, forbidden) {
				t.Errorf("forbidden release symbol %q: %s", forbidden, line)
			}
		}
	}
}
