package diffsurface

import (
	"context"
	"errors"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/git"
)

func TestPRGenerationPreservesEditsUntilExplicitApply(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	task := &prGenerationTask{root: "root", cancel: cancel, done: make(chan prGenerated, 1)}
	w := Workbench{Remote: &remoteWorkflowState{Mode: "pr", Title: "My edit", Body: "My body", Generation: task}}
	task.done <- prGenerated{content: git.GeneratedPRContent{Title: "Generated", Body: "Suggested"}}
	finishPRGeneration(Model{Workbench: &w, WorkspaceRoot: "root"})
	if ctx.Err() == nil || w.Remote.Title != "My edit" || w.Remote.Body != "My body" || w.Remote.Generated == nil {
		t.Fatal(w.Remote)
	}
	w.applyGeneratedPR()
	if w.Remote.Title != "Generated" || w.Remote.Body != "Suggested" || w.Remote.Generated != nil {
		t.Fatal(w.Remote)
	}
}
func TestPRGenerationFailureAndCancellationKeepDraft(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		_, cancel := context.WithCancel(context.Background())
		task := &prGenerationTask{root: "root", cancel: cancel, done: make(chan prGenerated, 1)}
		w := Workbench{Remote: &remoteWorkflowState{Title: "Keep", Body: "Keep body", Generation: task}}
		if canceled {
			w.cancelPRGeneration()
		}
		task.done <- prGenerated{err: errors.New("fixture failure")}
		finishPRGeneration(Model{Workbench: &w, WorkspaceRoot: "root"})
		if w.Remote.Title != "Keep" || w.Remote.Body != "Keep body" || w.Remote.Generated != nil {
			t.Fatal(w.Remote)
		}
		if !canceled && w.Remote.GenerationError == "" {
			t.Fatal("failure hidden")
		}
	}
}
func TestLargeFileAcknowledgementRevocationInvalidatesReview(t *testing.T) {
	fg, muted := ui.RGB(24, 24, 24), ui.RGB(120, 120, 120)
	w := &Workbench{Changes: []git.Change{{Path: "dir/huge.bin", Index: 'M', Worktree: ' '}},
		Commit:      &commitSelectionState{LargeAcknowledged: true, Excluded: map[string]bool{}, Plan: &git.SelectedCommitPlan{}, StackPlan: &git.StackPlan{}},
		ReviewToken: "selected", ReviewText: "patch"}
	w.Commit.LargeFiles = []git.LargeFile{{Path: "dir/huge.bin", Size: 200 << 20}}
	text := ""
	m := Model{Workbench: w, WorkspaceRoot: "workspace", Text: &text}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			paintCommitSelection(c, m, fg, muted)
		})
	}, 720, 700)
	if err := tt.Click(word("保留仍被勾选的大文件；我已了解风险，并愿意继续", "Keep checked large files; I understand they may be rejected")); err != nil {
		t.Fatalf("acknowledgement checkbox not reachable: %v", err)
	}
	if w.Commit.LargeAcknowledged {
		t.Fatal("revoked acknowledgement flag did not clear")
	}
	if w.Commit.Plan != nil || w.Commit.StackPlan != nil || w.ReviewToken != "" || w.ReviewText != "" {
		t.Fatal("revoked acknowledgement retained review")
	}
}
