package diffsurface

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
)

func boundTestColors() (ui.Color, ui.Color) {
	return ui.RGB(24, 24, 24), ui.RGB(120, 120, 120)
}

func commitSelectionWorkbench() (*Workbench, *commitSelectionState) {
	w := &Workbench{Changes: []git.Change{{Path: "dir/alpha.go", Index: 'M', Worktree: ' '}}}
	s := w.commitState()
	s.Plan = &git.SelectedCommitPlan{Summary: "prepared"}
	w.ReviewToken, w.ReviewText = "selected", "patch"
	return w, s
}

func reviewInvalidated(t *testing.T, w *Workbench, s *commitSelectionState) {
	t.Helper()
	if s.Plan != nil || w.ReviewToken != "" || w.ReviewText != "" {
		t.Fatalf("review was not invalidated: plan=%v token=%q text=%q", s.Plan, w.ReviewToken, w.ReviewText)
	}
}

// Toggling a file checkbox must flip its exclusion and invalidate the review.
func TestCommitCheckboxToggleUpdatesSelection(t *testing.T) {
	w, s := commitSelectionWorkbench()
	s.LargeAcknowledged = true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitCheckbox(c, w, "dir/alpha.go")
		})
	}, 400, 80)
	if err := tt.Click("dir/alpha.go"); err != nil {
		t.Fatalf("checkbox not reachable: %v", err)
	}
	if !s.Excluded["dir/alpha.go"] {
		t.Fatal("file checkbox click did not exclude the path")
	}
	if s.LargeAcknowledged {
		t.Fatal("file checkbox click kept the large-file acknowledgement")
	}
	reviewInvalidated(t, w, s)
}

// Toggling a file checkbox back on must include it again.
func TestCommitCheckboxToggleIncludesExcludedPath(t *testing.T) {
	w, s := commitSelectionWorkbench()
	s.Excluded["dir/alpha.go"] = true
	s.LargeAcknowledged = true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitCheckbox(c, w, "dir/alpha.go")
		})
	}, 400, 80)
	if err := tt.Click("dir/alpha.go"); err != nil {
		t.Fatalf("checkbox not reachable: %v", err)
	}
	if s.Excluded["dir/alpha.go"] {
		t.Fatal("file checkbox click did not re-include the path")
	}
	reviewInvalidated(t, w, s)
}

// The large-file acknowledgement checkbox must invalidate the review.
func TestLargeAcknowledgementInvalidatesReview(t *testing.T) {
	fg, muted := boundTestColors()
	w, s := commitSelectionWorkbench()
	s.LargeFiles = []git.LargeFile{{Path: "dir/huge.bin", Size: 200 << 20}}
	text := ""
	m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitSelection(c, m, fg, muted)
		})
	}, 720, 700)
	label := word("保留仍被勾选的大文件；我已了解风险，并愿意继续", "Keep checked large files; I understand they may be rejected")
	if err := tt.Click(label); err != nil {
		t.Fatalf("acknowledgement checkbox not reachable: %v", err)
	}
	if !s.LargeAcknowledged {
		t.Fatal("acknowledgement checkbox click did not set LargeAcknowledged")
	}
	reviewInvalidated(t, w, s)
}

// Typing a commit message must invalidate the prepared review.
func TestCommitMessageEditInvalidatesReview(t *testing.T) {
	fg, muted := boundTestColors()
	w, s := commitSelectionWorkbench()
	text := ""
	m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
	ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitSelection(c, m, fg, muted)
		})
	}, 720, 700).Type("wip: adjust")
	if w.Message != "wip: adjust" {
		t.Fatalf("message input lost typed text: %q", w.Message)
	}
	reviewInvalidated(t, w, s)
}

// Checking "commit on a new branch" must invalidate the review.
func TestCommitMakeBranchToggleInvalidatesReview(t *testing.T) {
	fg, muted := boundTestColors()
	w, s := commitSelectionWorkbench()
	text := ""
	m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitSelection(c, m, fg, muted)
		})
	}, 720, 700)
	if err := tt.Click(word("在新分支提交", "Commit on a new branch")); err != nil {
		t.Fatalf("branch checkbox not reachable: %v", err)
	}
	if !s.MakeBranch {
		t.Fatal("new-branch checkbox click did not set MakeBranch")
	}
	reviewInvalidated(t, w, s)
}

// tabFocus advances the focus chain n steps; inputs carry no findable label,
// so tests reach them through focus order instead of tt.Click.
func tabFocus(tt *ui.Tester, n int) {
	for i := 0; i < n; i++ {
		tt.Key(0, ui.KeyTab)
	}
}

// Typing a branch name must invalidate the review.
func TestCommitNewBranchEditInvalidatesReview(t *testing.T) {
	fg, muted := boundTestColors()
	w, s := commitSelectionWorkbench()
	s.MakeBranch = true
	text := ""
	m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitSelection(c, m, fg, muted)
		})
	}, 720, 700)
	// Focus order: select-all, select-none, message (AutoFocus), MakeBranch,
	// NewBranch. The message input already holds focus; two tabs reach the
	// branch input.
	tabFocus(tt, 2)
	tt.Type("fix/inputs")
	if s.NewBranch != "fix/inputs" {
		t.Fatalf("branch input lost typed text: %q", s.NewBranch)
	}
	reviewInvalidated(t, w, s)
}

// Toggling the PR switch must invalidate the review.
func TestCommitWantPRToggleInvalidatesReview(t *testing.T) {
	fg, muted := boundTestColors()
	w, s := commitSelectionWorkbench()
	text := ""
	m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitSelection(c, m, fg, muted)
		})
	}, 720, 700)
	if err := tt.Click(word("准备提交、推送并创建 PR", "Prepare commit, push and PR")); err != nil {
		t.Fatalf("PR checkbox not reachable: %v", err)
	}
	if !s.WantPR {
		t.Fatal("PR checkbox click did not set WantPR")
	}
	reviewInvalidated(t, w, s)
}

// Editing any PR field must invalidate the review.
func TestCommitPRFieldEditsInvalidateReview(t *testing.T) {
	fg, muted := boundTestColors()
	newWorkbench := func() *Workbench {
		w, s := commitSelectionWorkbench()
		s.WantPR, s.PRDraft = true, true
		return w
	}
	mk := func(w *Workbench) *ui.Tester {
		text := ""
		m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
		return ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				paintCommitSelection(c, m, fg, muted)
			})
		}, 720, 760)
	}
	t.Run("draft", func(t *testing.T) {
		w := newWorkbench()
		s := w.Commit
		tt := mk(w)
		if err := tt.Click(word("创建草稿 PR（取消勾选为普通 PR）", "Create draft PR (uncheck for ready PR)")); err != nil {
			t.Fatalf("draft checkbox not reachable: %v", err)
		}
		if s.PRDraft {
			t.Fatal("draft checkbox click did not clear PRDraft")
		}
		reviewInvalidated(t, w, s)
	})
	t.Run("title", func(t *testing.T) {
		w := newWorkbench()
		s := w.Commit
		tt := mk(w)
		// From the AutoFocus'd message input: MakeBranch, WantPR, PRDraft,
		// then PRTitle.
		tabFocus(tt, 4)
		tt.Type("feat: inputs")
		if s.PRTitle != "feat: inputs" {
			t.Fatalf("title input lost typed text: %q", s.PRTitle)
		}
		reviewInvalidated(t, w, s)
	})
	t.Run("body", func(t *testing.T) {
		w := newWorkbench()
		s := w.Commit
		tt := mk(w)
		tabFocus(tt, 5)
		tt.Type("desc")
		if s.PRBody != "desc" {
			t.Fatalf("body input lost typed text: %q", s.PRBody)
		}
		reviewInvalidated(t, w, s)
	})
}

// Editing any remote-workflow PR field must drop the prepared plan.
func TestRemoteWorkflowFieldEditsClearPlan(t *testing.T) {
	fg, muted := boundTestColors()
	newWorkbench := func() (*Workbench, *remoteWorkflowState) {
		w := &Workbench{Changes: []git.Change{{Path: "a.go", Index: 'M', Worktree: ' '}}}
		w.Remote = &remoteWorkflowState{Mode: "pr", Draft: true, Title: "t", Body: "b", Plan: &git.StackPlan{}, PR: &git.PullRequestPlan{}}
		return w, w.Remote
	}
	mk := func(w *Workbench) *ui.Tester {
		text := ""
		m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
		return ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				paintRemoteWorkflow(c, m, fg, muted)
			})
		}, 720, 640)
	}
	assertCleared := func(t *testing.T, s *remoteWorkflowState) {
		t.Helper()
		if s.Plan != nil || s.PR != nil {
			t.Fatalf("editing did not clear the prepared plan: plan=%v pr=%v", s.Plan, s.PR)
		}
	}
	t.Run("draft", func(t *testing.T) {
		w, s := newWorkbench()
		tt := mk(w)
		if err := tt.Click(word("创建草稿 PR（取消勾选为普通 PR）", "Create draft PR (uncheck for ready PR)")); err != nil {
			t.Fatalf("draft checkbox not reachable: %v", err)
		}
		assertCleared(t, s)
	})
	t.Run("title", func(t *testing.T) {
		w, s := newWorkbench()
		tt := mk(w)
		// Focus order: draft checkbox, title input, body area.
		tabFocus(tt, 2)
		tt.Type("x")
		assertCleared(t, s)
	})
	t.Run("body", func(t *testing.T) {
		w, s := newWorkbench()
		tt := mk(w)
		tabFocus(tt, 3)
		tt.Type("x")
		assertCleared(t, s)
	})
}
