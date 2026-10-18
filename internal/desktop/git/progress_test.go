package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitProgressTraceAndOutput(t *testing.T) {
	root, _, write := selectionRepo(t)
	write("a.txt", "selected\n")
	os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho fixture hook running\n"), 0700)
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	progress := NewProgress(root, "Commit", nil)
	ctx := WithProgress(context.Background(), progress)
	if _, e = CommitSelectedReviewed(ctx, root, "fixture progress", p); e != nil {
		t.Fatal(e)
	}
	s := progress.Snapshot()
	joined := strings.Join(s.Lines, "\n")
	if !strings.Contains(joined, "fixture hook running") || !strings.Contains(joined, "Running pre-commit") || !strings.Contains(joined, "Finished pre-commit") {
		t.Fatalf("missing real hook progress: %s", joined)
	}
}
func TestGitProgressBoundedAndAdvice(t *testing.T) {
	p := NewProgress("fixture", "Start", nil)
	for i := 0; i < 200; i++ {
		p.Line(strings.Repeat("x", 900))
	}
	p.Line("\x1b[31mhint: ignored\x1b[0m")
	s := p.Snapshot()
	if len(s.Lines) > 64 || len(s.Lines) == 0 {
		t.Fatal(len(s.Lines))
	}
	for _, line := range s.Lines {
		if len([]rune(line)) > 500 || strings.Contains(line, "hint:") {
			t.Fatal(line)
		}
	}
	p.Finish("Commit preserved; push failed", true)
	if s = p.Snapshot(); !s.Done || !s.Failed || s.Phase != "Commit preserved; push failed" {
		t.Fatal(s)
	}
}
func TestGitProgressCancellationRetainsFailureOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := NewProgress("fixture", "Start", nil)
	cmd := exec.CommandContext(ctx, "git", "--version")
	if _, e := observedCombinedOutput(WithProgress(ctx, p), cmd); e == nil {
		t.Fatal("cancel ignored")
	}
}
func TestGitProgressDismissedCompletionIgnoresLateEvents(t *testing.T) {
	p := NewProgress("fixture", "Preparing", nil)
	p.Finish("Canceled", false)
	p.Phase("Generating")
	p.Line("late output")
	p.Finish("Failed later", true)
	s := p.Snapshot()
	if s.Phase != "Canceled" || s.Failed || len(s.Lines) != 0 {
		t.Fatalf("late task repainted cancellation: %+v", s)
	}
}
func TestGitProgressPreservesUTF8(t *testing.T) {
	p := NewProgress("fixture", "Output", nil)
	w := &progressOutput{progress: p}
	w.Write([]byte("进度 正常\n"))
	w.finish()
	s := p.Snapshot()
	if len(s.Lines) != 1 || s.Lines[0] != "进度 正常" {
		t.Fatal(s.Lines)
	}
}
