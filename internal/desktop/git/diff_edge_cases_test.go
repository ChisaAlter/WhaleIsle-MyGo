package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitForTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestDiffUnbornHeadColorAlwaysAndTrailingBlankContext(t *testing.T) {
	dir := t.TempDir()
	runGitForTest(t, dir, "init")
	runGitForTest(t, dir, "config", "color.diff", "always")

	fileA := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(fileA, []byte("line1\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, dir, "add", "a.txt")
	// Modify working tree while HEAD is still unborn so both staged and unstaged diffs exist.
	if err := os.WriteFile(fileA, []byte("line1\nline2\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diffOut, err := DiffHEAD(dir, "")
	if err != nil {
		t.Fatalf("DiffHEAD unborn failed: %v", err)
	}
	if strings.Contains(diffOut, "\x1b[") {
		t.Fatalf("DiffHEAD leaked ANSI color escape codes: %q", diffOut)
	}
	// Ensure the second diff --git header begins on its own line.
	if idx := strings.Index(diffOut, "diff --git"); idx < 0 {
		t.Fatalf("missing diff header: %q", diffOut)
	}
	if strings.Contains(diffOut, "+line1diff --git") || strings.Contains(diffOut, "+diff --git") {
		t.Fatalf("unborn staged+unstaged diff concatenated without newline: %q", diffOut)
	}
	count := 0
	for _, line := range strings.Split(diffOut, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected 2 separate diff --git header lines for unborn staged+unstaged, got %d:\n%s", count, diffOut)
	}

	// IndexReview must also strip ANSI colors under color.diff=always.
	idxText, _, err := IndexReview(dir)
	if err != nil {
		t.Fatalf("IndexReview failed: %v", err)
	}
	if strings.Contains(idxText, "\x1b[") {
		t.Fatalf("IndexReview leaked ANSI codes: %q", idxText)
	}

	// Commit baseline with a trailing blank line, then edit earlier line and
	// verify trailing blank context line (" ") is preserved by Diff and CompareBranch.
	if err := os.WriteFile(fileA, []byte("first\nsecond\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, dir, "add", "a.txt")
	runGitForTest(t, dir, "commit", "-m", "initial")

	if err := os.WriteFile(fileA, []byte("FIRST\nsecond\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workingDiff, err := Diff(dir, "a.txt")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if strings.Contains(workingDiff, "\x1b[") {
		t.Fatalf("Diff leaked ANSI codes: %q", workingDiff)
	}
	lines := strings.Split(workingDiff, "\n")
	if len(lines) == 0 || lines[len(lines)-1] != " " {
		t.Fatalf("expected trailing blank context line \" \" preserved, got last line %q in:\n%s", lines[len(lines)-1], workingDiff)
	}

	// Also test CompareBranch preserves trailing blank context line.
	runGitForTest(t, dir, "checkout", "-b", "feature")
	runGitForTest(t, dir, "add", "a.txt")
	runGitForTest(t, dir, "commit", "-m", "change")
	baseBranch := "master"
	if _, err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "refs/heads/main").CombinedOutput(); err == nil {
		baseBranch = "main"
	}
	cmpDiff, err := CompareBranch(context.Background(), dir, "a.txt", baseBranch)
	if err != nil {
		t.Fatalf("CompareBranch failed: %v", err)
	}
	cmpLines := strings.Split(cmpDiff, "\n")
	if len(cmpLines) == 0 || cmpLines[len(cmpLines)-1] != " " {
		t.Fatalf("CompareBranch stripped trailing blank context line, got last=%q", cmpLines[len(cmpLines)-1])
	}
}
