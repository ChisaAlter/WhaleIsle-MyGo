package diffsurface

import (
	"context"
	"testing"
	"whaleisle/internal/desktop/git"
)

func TestRemoteWorkflowCanceledPreparationCannotRestoreReview(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	task := &remotePreparation{root: "old", cancel: cancel, done: make(chan remotePrepared, 1)}
	w := Workbench{Remote: &remoteWorkflowState{Task: task}}
	w.cancelRemotePreparation()
	task.done <- remotePrepared{}
	finishRemotePreparation(Model{Workbench: &w, WorkspaceRoot: "old"})
	if ctx.Err() == nil || w.Remote.Plan != nil {
		t.Fatal("cancelled completion restored review")
	}
}
func TestRemoteWorkflowHistoryKeepsCompletedStagesOnlyForSameHead(t *testing.T) {
	old := &git.StackResult{Committed: true, CommitSHA: "first"}
	next := &git.StackResult{Pushed: true, CommitSHA: "first"}
	merged := mergeStackHistory(old, next)
	if !merged.Committed || !merged.Pushed || next.Committed {
		t.Fatal(merged, next)
	}
	other := mergeStackHistory(old, &git.StackResult{Pushed: true, CommitSHA: "second"})
	if other.Committed {
		t.Fatal("history crossed HEAD", other)
	}
	if mergeStackHistory(old, &git.StackResult{}) != old {
		t.Fatal("preflight failure removed completed commit")
	}
}
