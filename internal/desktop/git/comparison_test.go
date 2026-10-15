package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func diffFixture(t *testing.T) (string, func(...string) string, func(string, string)) {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-b", "baseline")
	run("config", "user.name", "Diff Fixture")
	run("config", "user.email", "fixture@example.invalid")
	run("config", "commit.gpgsign", "false")
	write("a.txt", "initial\n")
	write("b.txt", "initial\n")
	run("add", ".")
	run("commit", "-m", "initial")
	return root, run, write
}

func TestCompareBranchUsesMergeBaseWithoutChangingTree(t *testing.T) {
	root, run, write := diffFixture(t)
	run("switch", "-c", "feature")
	write("a.txt", "feature\n")
	run("add", "a.txt")
	run("commit", "-m", "feature")
	run("switch", "baseline")
	write("b.txt", "baseline-only\n")
	run("add", "b.txt")
	run("commit", "-m", "baseline advance")
	run("switch", "feature")
	write("a.txt", "staged\n")
	run("add", "a.txt")
	write("a.txt", "worktree\n")
	write("new.txt", "untracked\n")
	before := run("status", "--porcelain=v1")
	head := run("rev-parse", "HEAD")
	out, err := CompareBranch(context.Background(), root, "", "baseline")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "+feature") || strings.Contains(out, "baseline-only") || strings.Contains(out, "+staged") || strings.Contains(out, "+worktree") || strings.Contains(out, "untracked") {
		t.Fatalf("wrong comparison: %s", out)
	}
	if run("status", "--porcelain=v1") != before || run("rev-parse", "HEAD") != head || run("branch", "--show-current") != "feature" {
		t.Fatal("comparison changed repository")
	}
	out, err = CompareBranch(context.Background(), root, "b.txt", "baseline")
	if err != nil || out != "" {
		t.Fatalf("path scope: %q %v", out, err)
	}
}

func TestCompareBranchRejectsUnsafeAndMissingRefs(t *testing.T) {
	root, _, _ := diffFixture(t)
	for _, ref := range []string{"", "--output=oops", "baseline~1", "baseline...HEAD", "missing", "baseline\n"} {
		if _, err := CompareBranch(context.Background(), root, "", ref); err == nil {
			t.Errorf("accepted %q", ref)
		}
	}
	if _, err := CompareBranch(context.Background(), root, "../outside", "baseline"); err == nil {
		t.Fatal("outside path accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CompareBranch(ctx, root, "", "baseline"); err == nil {
		t.Fatal("cancellation ignored")
	}
}
