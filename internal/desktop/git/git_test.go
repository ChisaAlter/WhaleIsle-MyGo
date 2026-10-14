package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceSnapshotClean(t *testing.T) {
	root := initTempRepo(t)
	snap, err := WorkspaceSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Branch == "" {
		t.Fatal("empty branch")
	}
	if snap.Dirty {
		t.Fatal("expected clean tree")
	}
	if FormatBadge(snap) != snap.Branch {
		t.Fatalf("badge = %q", FormatBadge(snap))
	}
}

func TestWorkspaceSnapshotDirty(t *testing.T) {
	root := initTempRepo(t)
	if err := os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap, err := WorkspaceSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Dirty {
		t.Fatal("expected dirty")
	}
	if FormatBadge(snap) != snap.Branch+" •" {
		t.Fatalf("badge = %q", FormatBadge(snap))
	}
}

func TestWorkspaceSnapshotNotRepo(t *testing.T) {
	_, err := WorkspaceSnapshot("")
	if err != ErrNotRepo {
		t.Fatalf("empty err = %v", err)
	}
	_, err = workspaceSnapshot(t.TempDir(), func(string, string, ...string) ([]byte, error) {
		return nil, exec.ErrNotFound
	})
	if err != ErrNotRepo {
		t.Fatalf("mock err = %v, want ErrNotRepo", err)
	}
}

func TestDirtyFromShortStatus(t *testing.T) {
	if dirtyFromShortStatus([]byte("## main")) {
		t.Fatal("single line should be clean")
	}
	if !dirtyFromShortStatus([]byte("## main\n M foo.go")) {
		t.Fatal("expected dirty")
	}
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.local")
	runGit(t, dir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGitCLIArgs(t *testing.T) {
	if got := gitCLIArgs("pull"); len(got) != 1 || got[0] != "pull" {
		t.Fatalf("pull args = %v", got)
	}
	if got := gitCLIArgs("fetch", "--prune"); len(got) != 2 || got[1] != "--prune" {
		t.Fatalf("fetch args = %v", got)
	}
}

func TestRunGitCommandMockRunner(t *testing.T) {
	root := initTempRepo(t)
	var seenDir string
	var seenArgs []string
	mock := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		seenDir = dir
		seenArgs = append([]string(nil), args...)
		return []byte("mock out\n"), nil
	}
	out, err := runGitCommand(root, time.Second, mock, "pull")
	if err != nil {
		t.Fatal(err)
	}
	if out != "mock out" {
		t.Fatalf("out = %q", out)
	}
	if seenDir != filepath.Clean(root) {
		t.Fatalf("dir = %q", seenDir)
	}
	if len(seenArgs) != 1 || seenArgs[0] != "pull" {
		t.Fatalf("args = %v", seenArgs)
	}
}

func TestRunGitCommandNotRepo(t *testing.T) {
	_, err := runGitCommand("", time.Second, combinedExecRunner, "pull")
	if err != ErrNotRepo {
		t.Fatalf("err = %v", err)
	}
}

func TestDiffNotRepo(t *testing.T) {
	_, err := Diff("")
	if err != ErrNotRepo {
		t.Fatalf("empty err = %v", err)
	}
}

func TestRunGitDiffNotInsideWorkTree(t *testing.T) {
	root := t.TempDir()
	mock := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		return nil, exec.ErrNotFound
	}
	_, err := runGitDiff(root, time.Second, mock)
	if err != ErrNotRepo {
		t.Fatalf("err = %v", err)
	}
}

func TestDiffMockRunner(t *testing.T) {
	root := initTempRepo(t)
	var seenArgs []string
	mock := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		seenArgs = append([]string(nil), args...)
		return []byte("--- a\n+++ b\n"), nil
	}
	out, err := runGitDiff(root, time.Second, mock, "foo.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "+++ b") {
		t.Fatalf("out = %q", out)
	}
	if len(seenArgs) < 3 || seenArgs[0] != "diff" || seenArgs[1] != "--" || seenArgs[2] != "foo.go" {
		t.Fatalf("args = %v", seenArgs)
	}
}

func TestDiffDirtyWorktree(t *testing.T) {
	root := initTempRepo(t)
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Diff(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "README") {
		t.Fatalf("out = %q", out)
	}
}

func TestRunGitCommandTimeout(t *testing.T) {
	root := initTempRepo(t)
	mock := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	_, err := runGitCommand(root, 20*time.Millisecond, mock, "push")
	if err == nil {
		t.Fatal("expected timeout")
	}
}

func TestPullPushWithMockGitOnPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PATH git.cmd shim is Windows-oriented")
	}
	bin := t.TempDir()
	script := filepath.Join(bin, "git.cmd")
	const body = `@echo off
echo MOCK %*
exit /b 0
`
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	root := initTempRepo(t)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := Pull(root)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !strings.Contains(out, "MOCK pull") {
		t.Fatalf("pull out = %q", out)
	}
	out, err = Push(root)
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if !strings.Contains(out, "MOCK push") {
		t.Fatalf("push out = %q", out)
	}
	out, err = Fetch(root)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !strings.Contains(out, "MOCK fetch") {
		t.Fatalf("fetch out = %q", out)
	}
}

func TestCommitAllEmptyMessage(t *testing.T) {
	root := initTempRepo(t)
	_, err := CommitAll(root, "  ")
	if err != ErrEmptyCommitMessage {
		t.Fatalf("err = %v", err)
	}
}

func TestCommitAllMockRunner(t *testing.T) {
	root := initTempRepo(t)
	if err := os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	mock := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		return []byte("ok"), nil
	}
	out, err := commitAll(root, "wip", time.Second, mock)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Fatalf("out = %q", out)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v", calls)
	}
	if calls[0][0] != "add" || calls[1][0] != "commit" || calls[1][2] != "wip" {
		t.Fatalf("calls = %v", calls)
	}
}
