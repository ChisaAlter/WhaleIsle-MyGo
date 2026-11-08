package diffsurface

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/diff"
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

func TestUnifiedDiffRowsSelectionExcludesGutter(t *testing.T) {
	rows := []patchRow{
		{Old: "-oldLine()", New: "+newLine()", OldNumber: 10, NewNumber: 10},
		{Old: " keepLine()", New: " keepLine()", OldNumber: 11, NewNumber: 11},
	}
	for _, wrap := range []bool{false, true} {
		tt := ui.NewTester(func(c *ui.Context) {
			paintUnifiedRows(c, rows, diff.Style{FontSize: 12, FG: ui.RGB(0, 0, 0), Add: ui.RGB(0, 0, 0), Del: ui.RGB(0, 0, 0), Muted: ui.RGB(0, 0, 0)}, wrap, hunkFold{})
		}, 600, 200)
		if err := tt.Click("newLine()"); err != nil {
			t.Fatalf("wrap=%v click: %v", wrap, err)
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Key(ui.Cmd, ui.KeyC)
		got := tt.Clipboard()
		if !strings.Contains(got, "oldLine()") || !strings.Contains(got, "newLine()") || !strings.Contains(got, "keepLine()") {
			t.Fatalf("wrap=%v expected diff code lines in clipboard, got %q", wrap, got)
		}
		if strings.Contains(got, "10") || strings.Contains(got, "11") {
			t.Fatalf("wrap=%v expected line numbers excluded from clipboard, got %q", wrap, got)
		}
	}
}

