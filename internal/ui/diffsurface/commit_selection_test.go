package diffsurface

import (
	"context"
	"testing"
	"whaleisle/internal/desktop/git"
)

func TestCommitSelectionExcludesExplicitlyAndDoesNotDropLargeFiles(t *testing.T) {
	w := Workbench{Changes: []git.Change{{Path: "small"}, {Path: "large"}}, Commit: &commitSelectionState{Excluded: map[string]bool{}}}
	if len(w.selectedCommitPaths()) != 2 {
		t.Fatal("files implicitly omitted")
	}
	w.Commit.Excluded["large"] = true
	if paths := w.selectedCommitPaths(); len(paths) != 1 || paths[0] != "small" {
		t.Fatal(paths)
	}
}
func TestCommitSelectionCancelDiscardsLatePreparation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := Workbench{Commit: &commitSelectionState{Task: &commitPreparation{cancel: cancel, done: make(chan commitPrepared, 1)}, Plan: &git.SelectedCommitPlan{Summary: "old"}}}
	task := w.Commit.Task
	w.cancelCommitPreparation()
	if ctx.Err() == nil || w.Commit.Task != nil || w.Commit.Plan != nil {
		t.Fatal("cancel did not clear ownership")
	}
	task.done <- commitPrepared{plan: git.SelectedCommitPlan{Summary: "late"}}
	finishCommitPreparation(Model{Workbench: &w, WorkspaceRoot: "new"})
	if w.Commit.Plan != nil {
		t.Fatal("late completion restored a dismissed plan")
	}
}
