package diffsurface

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestLoadDiffNoWorkspace(t *testing.T) {
	i18n.SetLocale("en")
	_, errMsg := LoadDiff("", "")
	if errMsg != i18n.T(i18n.KeyDiffNoWorkspace) {
		t.Fatalf("errMsg = %q", errMsg)
	}
}

func TestLoadDiffEmptyCleanRepo(t *testing.T) {
	i18n.SetLocale("en")
	root := initTempRepo(t)
	_, errMsg := LoadDiff(root, "")
	if errMsg != i18n.T(i18n.KeyDiffEmpty) {
		t.Fatalf("errMsg = %q", errMsg)
	}
}

func TestLoadDiffWithChanges(t *testing.T) {
	root := initTempRepo(t)
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	text, errMsg := LoadDiff(root, "README")
	if errMsg != "" {
		t.Fatalf("errMsg = %q", errMsg)
	}
	if !strings.Contains(text, "README") {
		t.Fatalf("text = %q", text)
	}
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.local")
	runGit(t, dir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
