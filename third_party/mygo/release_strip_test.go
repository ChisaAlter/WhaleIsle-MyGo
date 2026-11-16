package mygo_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseBuildStripsTestHooks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping build test in short mode")
	}

	tmpDir := t.TempDir()

	mygoDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("failed to resolve mygo dir: %v", err)
	}
	mygoDir = filepath.ToSlash(mygoDir)

	goModContent := fmt.Sprintf(`module testrelease

go 1.27

replace github.com/egoist/mygo => %s

require github.com/egoist/mygo v0.0.0
`, mygoDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	mainContent := `package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	_ = mygo.Version
	_ = ui.Rect{}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	tidyCmd.Env = os.Environ()
	if tidyOut, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\nOutput:\n%s", err, string(tidyOut))
	}

	binPath := filepath.Join(tmpDir, "testapp.exe")

	buildCmd := exec.Command("go", "build", "-tags", "release", "-trimpath", "-o", binPath, ".")
	buildCmd.Dir = tmpDir
	buildCmd.Env = os.Environ()
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build -tags release failed: %v\nOutput:\n%s", err, string(out))
	}

	nmCmd := exec.Command("go", "tool", "nm", binPath)
	nmCmd.Dir = tmpDir
	nmCmd.Env = os.Environ()
	nmOut, err := nmCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go tool nm failed: %v\nOutput:\n%s", err, string(nmOut))
	}

	forbiddenSymbols := []string{
		"TestClick",
		"TestAccessibility",
		"DriveWindow",
		"WindowDriver",
		"NewTester",
		"testCountersEnabled",
		"testFrameCountersEnabled",
	}

	nmLines := strings.Split(string(nmOut), "\n")
	for _, sym := range forbiddenSymbols {
		for _, line := range nmLines {
			if strings.Contains(line, sym) {
				t.Errorf("found forbidden test symbol %q in release binary:\n  %s", sym, line)
			}
		}
	}
}
