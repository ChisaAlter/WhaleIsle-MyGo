package diffsurface

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

// These Git commands execute only when root later runs tests. Every mutation
// is confined to t.TempDir, without remotes, hooks, signing or account access.
func captionFixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	hooks := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		flags := []string{"-c", "user.name=Caption Fixture", "-c", "user.email=fixture@local.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + hooks}
		cmd := exec.Command("git", append(flags, args...)...)
		cmd.Dir = root
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(entry), "GIT_") {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated git %v: %v: %s", args, err, output)
		}
	}
	run("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "--", "tracked.txt")
	run("commit", "-m", "isolated fixture")
	return root
}
func awaitCaptionFixture(t *testing.T, tt *ui.Tester, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if ready() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("isolated caption fixture did not settle")
}

func TestCaptionInitializeIsOneGestureAndCapturesItsRoot(t *testing.T) {
	root, next := t.TempDir(), t.TempDir()
	current := root
	w := &Workbench{}
	accepted := false
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Workbench: w, WorkspaceRoot: current, CaptionAdmission: func() bool { return true }}
		if ui.Button(c, "Initialize captured root").Clicked() {
			accepted = CaptionAction(c, m, "", "init", true)
		}
		CaptionActivity(c, m)
	}, 700, 500)
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatal("painting initialized a repository")
	}
	if err := tt.Click("Initialize captured root"); err != nil {
		t.Fatal(err)
	}
	current = next
	awaitCaptionFixture(t, tt, func() bool { return w.Operation == nil })
	if !accepted {
		t.Fatal("one gesture did not admit initialization")
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Fatal("captured root was not initialized", err)
	}
	if _, err := os.Stat(filepath.Join(next, ".git")); !os.IsNotExist(err) {
		t.Fatal("initialization crossed into a replacement cwd")
	}
	if w.CaptionReviewOpen() || w.Commit != nil && w.Commit.Task != nil {
		t.Fatal("Initialize inserted a second review or preparation")
	}
}

func TestCaptionGlobalReviewUsesSameWorkbenchAcrossPageAndRetiresOwner(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	root := captionFixtureRepo(t)
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &Workbench{}
	defer w.ClosePane()
	admitted, global := true, true
	m := Model{Workbench: w, WorkspaceRoot: root, CaptionAdmission: func() bool { return admitted }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Text(c, "Current global page")
		if ui.Button(c, "Open global commit").Clicked() {
			if !CaptionAction(c, m, "explicit fixture message", "commit_push", true) {
				t.Error("global gesture rejected")
			}
		}
		if !global {
			paintWorkbench(c, m, c.Theme().Text, c.Theme().TextMuted)
		}
		CaptionReview(c, m)
	}, 900, 900)
	if err := tt.Click("Open global commit"); err != nil {
		t.Fatal(err)
	}
	selection := w.Commit
	if !w.CaptionReviewOpen() || selection == nil || !tt.HasText("Commit changes") || !tt.HasText("tracked.txt") {
		t.Fatal("review was hidden behind the global page", tt.Texts())
	}
	if w.Operation != nil || selection.Task != nil || selection.Plan != nil || w.Remote != nil {
		t.Fatal("opening the global modal automatically prepared or executed an action")
	}
	global = false
	tt.Frame()
	count := 0
	for _, text := range tt.Texts() {
		if text == "Review commit and push…" {
			count++
		}
	}
	if w.Commit != selection || count != 1 {
		t.Fatalf("navigation cloned or duplicated the review: %d", count)
	}
	ctx, cancel := context.WithCancel(context.Background())
	task := &commitPreparation{root: root, cancel: cancel, done: make(chan commitPrepared, 1)}
	selection.Task = task
	admitted = false
	tt.Frame()
	if w.CaptionReviewOpen() || ctx.Err() == nil || selection.Task != nil || w.Operation != nil {
		t.Fatal("replacement owner retained executable review/preparation")
	}
	task.done <- commitPrepared{plan: git.SelectedCommitPlan{Summary: "retired result"}}
	finishCommitPreparation(m)
	if selection.Plan != nil {
		t.Fatal("retired preparation restored the review")
	}
}

func TestCaptionConfirmationAndGenerationRecheckAdmission(t *testing.T) {
	w := &Workbench{Root: "fixture", Commit: &commitSelectionState{Excluded: map[string]bool{}, Plan: &git.SelectedCommitPlan{}, StackPlan: &git.StackPlan{}}, Remote: &remoteWorkflowState{}}
	m := Model{Workbench: w, WorkspaceRoot: "fixture", CaptionAdmission: func() bool { return false }}
	ui.NewTester(func(c *ui.Context) {
		confirmSelectedCommit(c, m)
		beginCommitPreparation(c, m, true, true)
		beginRemotePreparation(c, m)
		beginPRGeneration(c, m)
	}, 400, 300)
	if w.Operation != nil || w.Commit.Task != nil || w.Remote.Task != nil || w.Remote.Generation != nil {
		t.Fatal("revoked admission reached an operation/model/preparation")
	}
}

func TestCaptionCleanCommitPushRoutesExistingPushReview(t *testing.T) {
	root := captionFixtureRepo(t)
	w := &Workbench{}
	defer w.ClosePane()
	if !w.RequestCaptionAction(root, "", "commit_push") || w.Remote == nil || w.Remote.Mode != "push" || w.Operation != nil || w.Remote.Task != nil {
		t.Fatal("clean default quick Push was trapped in empty file selection")
	}
}

func TestCaptionBranchPickerLoadsExistingOwnerAndCreateDoesNotRunUntilSubmit(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	root := captionFixtureRepo(t)
	w := &Workbench{Root: root}
	defer w.ClosePane()
	open := false
	m := Model{Workbench: w, WorkspaceRoot: root}
	tt := ui.NewTester(func(c *ui.Context) {
		anchor := ui.Button(c, "Choose branch")
		if anchor.Clicked() {
			open = !open
		}
		CaptionBranches(c, m, anchor, &open)
	}, 700, 650)
	if err := tt.Click("Choose branch"); err != nil {
		t.Fatal(err)
	}
	awaitCaptionFixture(t, tt, func() bool { return w.branchesRequest == nil && len(w.Branches) > 0 })
	if !tt.HasText("main") {
		t.Fatal("titlebar picker never consumed the branch list")
	}
	w.BranchQuery = "feature/fixture"
	tt.Frame()
	if err := tt.Click("Create branch feature/fixture…"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Create and switch") || w.Operation != nil {
		t.Fatal("opening create dialog mutated Git or hid its form")
	}
	tt.Key(0, ui.KeyEscape)
	if w.Operation != nil {
		t.Fatal("dismissing create dialog started a branch operation")
	}
}

func TestCaptionBranchReadFromRetiredRootCannotPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	request := &branchListRequest{root: "old", cancel: cancel, done: make(chan branchListResult, 1)}
	w := &Workbench{Root: "new", BranchesOpen: true, Branches: []git.Branch{{Name: "new-main"}}, branchesRequest: request}
	request.done <- branchListResult{branches: []git.Branch{{Name: "retired-main"}}}
	w.finishBranchList()
	if ctx.Err() == nil || w.branchesRequest != nil || len(w.Branches) != 1 || w.Branches[0].Name != "new-main" {
		t.Fatal("old root branch response entered the current picker")
	}
}
