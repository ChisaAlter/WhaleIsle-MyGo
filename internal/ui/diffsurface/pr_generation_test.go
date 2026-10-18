package diffsurface

import (
	"context"
	"errors"
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
	w := Workbench{Commit: &commitSelectionState{LargeAcknowledged: false, Plan: &git.SelectedCommitPlan{}, StackPlan: &git.StackPlan{}}, ReviewToken: "selected", ReviewText: "patch"}
	w.invalidateChangedLargeAcknowledgement(true)
	if w.Commit.Plan != nil || w.Commit.StackPlan != nil || w.ReviewToken != "" || w.ReviewText != "" {
		t.Fatal("revoked acknowledgement retained review")
	}
}
