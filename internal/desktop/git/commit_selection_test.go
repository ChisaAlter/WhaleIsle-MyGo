package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func selectionRepo(t *testing.T) (string, func(...string) string, func(string, string)) {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = root
		out, e := c.CombinedOutput()
		if e != nil {
			t.Fatalf("git %v: %s: %v", args, out, e)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(p, s string) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(root, p), []byte(s), 0600); e != nil {
			t.Fatal(e)
		}
	}
	run("init", "-b", "main")
	run("config", "user.name", "Fixture")
	run("config", "user.email", "fixture@example.invalid")
	run("config", "commit.gpgsign", "false")
	write("a.txt", "base\n")
	write("b.txt", "base\n")
	run("add", ".")
	run("commit", "-m", "base")
	return root, run, write
}
func TestSelectedCommitPreservesOtherIndexAndReviewsWorktree(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "selected working copy\n")
	write("b.txt", "unrelated staged\n")
	run("add", "b.txt")
	write("new.txt", "unselected\n")
	before := run("diff", "--cached", "--binary")
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(p.Patch, "+selected working copy") || strings.Contains(p.Summary, "b.txt") {
		t.Fatalf("wrong preview: %+v", p)
	}
	if got := run("diff", "--cached", "--binary"); got != before {
		t.Fatal("preparation changed real index")
	}
	result, e := CommitSelectedReviewed(context.Background(), root, "selected", p)
	if e != nil {
		t.Fatal(e)
	}
	if !result.Committed {
		t.Fatal("missing completion")
	}
	if got := run("show", "--format=", "--name-only", "HEAD"); got != "a.txt" {
		t.Fatalf("committed wrong paths: %s", got)
	}
	if got := run("diff", "--cached", "--binary"); got != before {
		t.Fatalf("unrelated staging changed: %s", got)
	}
	if got := run("diff", "--cached", "--", "a.txt"); got != "" {
		t.Fatalf("committed selection remains reverse staged: %s", got)
	}
	if got := run("status", "--porcelain"); !strings.Contains(got, "?? new.txt") {
		t.Fatalf("unselected changed: %s", got)
	}
}
func TestSelectedCommitRejectsChangedReview(t *testing.T) {
	for _, kind := range []string{"file", "index", "head", "repository"} {
		t.Run(kind, func(t *testing.T) {
			root, run, write := selectionRepo(t)
			write("a.txt", "selected\n")
			p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "")
			if e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "file":
				write("a.txt", "later\n")
			case "index":
				write("b.txt", "staged later\n")
				run("add", "b.txt")
			case "head":
				run("switch", "-c", "elsewhere")
			case "repository":
				root, _, _ = selectionRepo(t)
			}
			if _, e := CommitSelectedReviewed(context.Background(), root, "must not commit", p); e == nil {
				t.Fatal("stale review accepted")
			}
		})
	}
}
func TestSelectedCommitNewBranchAndConflict(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "selected\n")
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "feature/selected")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = CommitSelectedReviewed(context.Background(), root, "selected", p); e != nil {
		t.Fatal(e)
	}
	if got := run("branch", "--show-current"); got != "feature/selected" {
		t.Fatal(got)
	}
	write("a.txt", "later\n")
	if _, e = PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "main"); e == nil {
		t.Fatal("existing branch accepted")
	}
}
func TestSelectedCommitUnbornAndLiteralRename(t *testing.T) {
	root, run, write := selectionRepo(t)
	run("mv", "a.txt", "literal[1].txt")
	write("literal[1].txt", "renamed\n")
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"literal[1].txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = CommitSelectedReviewed(context.Background(), root, "rename", p); e != nil {
		t.Fatal(e)
	}
	if out := run("ls-tree", "--name-only", "HEAD"); strings.Contains(out, "a.txt") || !strings.Contains(out, "literal[1].txt") {
		t.Fatal(out)
	}
	root = t.TempDir()
	c := exec.Command("git", "init", "-b", "main", root)
	if out, e := c.CombinedOutput(); e != nil {
		t.Fatalf("%s %v", out, e)
	}
	for _, a := range [][]string{{"config", "user.name", "Fixture"}, {"config", "user.email", "fixture@example.invalid"}, {"config", "commit.gpgsign", "false"}} {
		c = exec.Command("git", a...)
		c.Dir = root
		if e := c.Run(); e != nil {
			t.Fatal(e)
		}
	}
	os.WriteFile(filepath.Join(root, "first"), []byte("first\n"), 0600)
	p, e = PrepareSelectedCommit(context.Background(), root, []string{"first"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = CommitSelectedReviewed(context.Background(), root, "first", p); e != nil {
		t.Fatal(e)
	}
}
func TestLargeFileBoundaryAndExclusion(t *testing.T) {
	root, _, _ := selectionRepo(t)
	for _, entry := range []struct {
		p string
		n int64
	}{{"equal.bin", 100 * 1024 * 1024}, {"large.bin", 100*1024*1024 + 1}} {
		f, e := os.Create(filepath.Join(root, entry.p))
		if e != nil {
			t.Fatal(e)
		}
		if e = f.Truncate(entry.n); e != nil {
			t.Fatal(e)
		}
		f.Close()
	}
	files, e := CheckLargeFiles(root)
	if e != nil {
		t.Fatal(e)
	}
	if len(files) != 1 || files[0].Path != "large.bin" {
		t.Fatalf("wrong threshold: %+v", files)
	}
	if _, e = PrepareSelectedCommit(context.Background(), root, nil, ""); e == nil {
		t.Fatal("empty selection accepted")
	}
}
func TestSelectedCommitHookFailurePreservesIndex(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "selected\n")
	write("b.txt", "staged\n")
	run("add", "b.txt")
	before := run("diff", "--cached", "--binary")
	head := run("rev-parse", "HEAD")
	os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho fixture-hook-failure >&2\nexit 1\n"), 0700)
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = CommitSelectedReviewed(context.Background(), root, "fail", p); e == nil {
		t.Fatal("hook failure ignored")
	}
	if got := run("diff", "--cached", "--binary"); got != before {
		t.Fatal("failed commit changed real index")
	}
	if got := run("rev-parse", "HEAD"); got != head {
		t.Fatal("failed commit moved HEAD")
	}
}

func TestSelectedCommitDeletedParentAndPartialStage(t *testing.T) {
	root, run, write := selectionRepo(t)
	os.Mkdir(filepath.Join(root, "dir"), 0700)
	write("dir/gone.txt", "old\n")
	run("add", "dir")
	run("commit", "-m", "directory")
	os.RemoveAll(filepath.Join(root, "dir"))
	write("a.txt", "staged\n")
	run("add", "a.txt")
	write("a.txt", "full working copy\n")
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"dir/gone.txt", "a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(p.Patch, "+full working copy") {
		t.Fatal("partial staged snapshot used")
	}
	if _, e = CommitSelectedReviewed(context.Background(), root, "selected", p); e != nil {
		t.Fatal(e)
	}
	if got := run("show", "HEAD:a.txt"); got != "full working copy" {
		t.Fatal(got)
	}
}
func TestSelectedCommitRefusesInProgressMerge(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "selected\n")
	os.WriteFile(filepath.Join(root, ".git", "MERGE_HEAD"), []byte(run("rev-parse", "HEAD")+"\n"), 0600)
	if _, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, ""); e == nil {
		t.Fatal("merge state silently committed")
	}
}
func TestSelectedCommitRejectsHookChangingReviewedIndex(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "selected\n")
	write("b.txt", "unselected\n")
	head := run("rev-parse", "HEAD")
	os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\ngit add b.txt\n"), 0700)
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = CommitSelectedReviewed(context.Background(), root, "must fail", p); e == nil {
		t.Fatal("hook silently included unreviewed file")
	}
	if got := run("rev-parse", "HEAD"); got != head {
		t.Fatal("hook changed reviewed commit")
	}
}
func TestSelectedCommitExcludesLargeUnselectedAndHonorsLock(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "selected\n")
	f, e := os.Create(filepath.Join(root, "large.bin"))
	if e != nil {
		t.Fatal(e)
	}
	f.Truncate(LargeFileLimit + 1)
	f.Close()
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "")
	if e != nil || len(p.LargeFiles) != 0 {
		t.Fatalf("excluded large file included: %+v %v", p, e)
	}
	lock := filepath.Join(root, ".git", "index.lock")
	os.WriteFile(lock, []byte("other process"), 0600)
	if _, e = CommitSelectedReviewed(context.Background(), root, "busy", p); e == nil {
		t.Fatal("concurrent index lock ignored")
	}
	os.Remove(lock)
	if _, e = CommitSelectedReviewed(context.Background(), root, "selected", p); e != nil {
		t.Fatal(e)
	}
	if got := run("status", "--porcelain"); !strings.Contains(got, "?? large.bin") {
		t.Fatal(got)
	}
}
func TestSelectedCommitScratchIndexDoesNotRunIndexHooks(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "selected\n")
	write("b.txt", "unselected\n")
	os.WriteFile(filepath.Join(root, ".git", "hooks", "post-index-change"), []byte("#!/bin/sh\nif test \"$FIXTURE_RECURSION\" != yes; then FIXTURE_RECURSION=yes git add b.txt; fi\n"), 0700)
	p, e := PrepareSelectedCommit(context.Background(), root, []string{"a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(p.Summary, "b.txt") {
		t.Fatal("scratch index hook added unselected file to preview")
	}
	result, e := CommitSelectedReviewed(context.Background(), root, "selected", p)
	if e == nil && !result.Committed {
		t.Fatal("no commit result")
	}
	if got := run("diff", "--cached", "--name-only"); got != "" {
		t.Fatalf("reconciliation hook changed actual index: %s", got)
	}
}
func TestSelectedCommitRejectsRenameAcrossWorkspaceBoundary(t *testing.T) {
	root, run, write := selectionRepo(t)
	os.Mkdir(filepath.Join(root, "sub"), 0700)
	write("sub/a.txt", "same name inside workspace\n")
	run("add", "sub")
	run("commit", "-m", "sub workspace")
	run("mv", "a.txt", "sub/moved.txt")
	if _, e := PrepareSelectedCommit(context.Background(), filepath.Join(root, "sub"), []string{"moved.txt"}, ""); e == nil {
		t.Fatal("cross-workspace rename interpreted as local path")
	}
}
