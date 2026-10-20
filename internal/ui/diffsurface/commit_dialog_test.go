package diffsurface

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

func commitDialogTester(t *testing.T) (*Workbench, *ui.Tester) {
	t.Helper()
	workbench := &Workbench{Changes: []git.Change{{Path: "dir/alpha.go", Index: 'M', Worktree: ' '}}}
	text := ""
	m := Model{Workbench: workbench, WorkspaceRoot: "workspace", Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			commitLauncher(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
			commitDialog(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
		})
	}, 720, 460)
	return workbench, tt
}

// TestCommitChecklistShowsDiffstat puts the git line counts on the row, so the
// checklist says how large each change is before the patch is opened.
func TestCommitChecklistShowsDiffstat(t *testing.T) {
	workbench := &Workbench{CommitOpen: true, Changes: []git.Change{
		{Path: "dir/alpha.go", Index: 'M', Worktree: ' ', Added: 12, Deleted: 3},
		{Path: "dir/beta.go", Index: 'A', Worktree: ' ', Added: 4},
	}}
	text := ""
	m := Model{Workbench: workbench, WorkspaceRoot: "workspace", Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			commitDialog(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
		})
	}, 720, 460)
	if !tt.HasText("+12") || !tt.HasText("-3") {
		t.Fatalf("edited row lost its diffstat: %v", tt.Texts())
	}
	if !tt.HasText("+4") {
		t.Fatalf("added row lost its diffstat: %v", tt.Texts())
	}
	for _, text := range tt.Texts() {
		if text == "-0" {
			t.Fatalf("deleted count printed for a file with no deletions: %v", tt.Texts())
		}
	}
}

func TestCommitFormStaysClosedUntilAsked(t *testing.T) {
	workbench, tt := commitDialogTester(t)
	if workbench.CommitOpen {
		t.Fatal("commit form opened without a request")
	}
	if tt.HasText(i18n.T(i18n.KeySurfacesGitCommitFiles)) {
		t.Fatalf("commit form rendered in the pane: %v", tt.Texts())
	}
	launcher := fmt.Sprintf(i18n.T(i18n.KeySurfacesGitCommitOpen), 1)
	if err := tt.Click(launcher); err != nil {
		t.Fatalf("launcher is not reachable: %v", err)
	}
	if !workbench.CommitOpen {
		t.Fatal("launcher did not open the commit form")
	}
	tt.Frame()
	if !tt.HasText(i18n.T(i18n.KeySurfacesGitCommitTitle)) {
		t.Fatalf("dialog lost its title: %v", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeySurfacesGitCommitFiles)) || !tt.HasText("dir/alpha.go") {
		t.Fatalf("dialog lost the staged checklist: %v", tt.Texts())
	}
}

// TestCommitDialogClosesOnItsCloseAction keeps the form dismissible without
// committing anything.
func TestCommitDialogClosesOnItsCloseAction(t *testing.T) {
	workbench, tt := commitDialogTester(t)
	launcher := fmt.Sprintf(i18n.T(i18n.KeySurfacesGitCommitOpen), 1)
	if err := tt.Click(launcher); err != nil {
		t.Fatalf("launcher is not reachable: %v", err)
	}
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeySurfacesGitClose)); err != nil {
		t.Fatalf("close action is not reachable: %v", err)
	}
	if workbench.CommitOpen {
		t.Fatal("close action left the form open")
	}
}
