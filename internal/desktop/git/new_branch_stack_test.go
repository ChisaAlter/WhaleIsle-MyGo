package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewBranchCommitPushStackUsesReviewedDestination(t *testing.T) {
	root, run, write := selectionRepo(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	if out, e := exec.Command("git", "init", "--bare", remote).CombinedOutput(); e != nil {
		t.Fatalf("%s %v", out, e)
	}
	run("remote", "add", "origin", remote)
	write("a.txt", "selected new branch\n")
	ctx := context.Background()
	commit, e := PrepareSelectedCommit(ctx, root, []string{"a.txt"}, "feature/new-stack")
	if e != nil {
		t.Fatal(e)
	}
	push, e := PreparePushForNewBranch(root, commit.NewBranch)
	if e != nil {
		t.Fatal(e)
	}
	if push.Branch != "feature/new-stack" || push.Target != push.Branch {
		t.Fatal(push)
	}
	plan, e := PrepareStack(ctx, root, &commit, "New branch change", &push, nil)
	if e != nil {
		t.Fatal(e)
	}
	if got := run("branch", "--show-current"); got != "main" {
		t.Fatal("review changed branch", got)
	}
	result, e := RunCommitStack(ctx, root, plan)
	if e != nil || !result.Committed || !result.Pushed || result.CreatedBranch != push.Branch {
		t.Fatal(result, e)
	}
	if got := run("rev-parse", "refs/remotes/origin/feature/new-stack"); got != result.CommitSHA {
		t.Fatal("wrong pushed commit", got)
	}
}

func TestNewBranchDestinationChangesStopBeforeCommit(t *testing.T) {
	root, run, write := selectionRepo(t)
	run("remote", "add", "origin", "https://github.com/example/project.git")
	write("a.txt", "new\n")
	ctx := context.Background()
	commit, e := PrepareSelectedCommit(ctx, root, []string{"a.txt"}, "feature/new")
	if e != nil {
		t.Fatal(e)
	}
	push, e := PreparePushForNewBranch(root, commit.NewBranch)
	if e != nil {
		t.Fatal(e)
	}
	plan, e := PrepareStack(ctx, root, &commit, "Change", &push, nil)
	if e != nil {
		t.Fatal(e)
	}
	before := run("rev-parse", "HEAD")
	run("remote", "set-url", "origin", "https://github.com/other/project.git")
	result, e := RunCommitStack(ctx, root, plan)
	if e == nil || result.Committed || run("rev-parse", "HEAD") != before || run("branch", "--show-current") != "main" {
		t.Fatal("destination drift accepted", result, e)
	}
}

func TestNewBranchPRReviewSurvivesOnlyReviewedCommitAndPush(t *testing.T) {
	root, run, write := selectionRepo(t)
	run("remote", "add", "origin", "https://github.com/example/project.git")
	write("a.txt", "reviewed feature\n")
	ctx := context.Background()
	commit, e := PrepareSelectedCommit(ctx, root, []string{"a.txt"}, "feature/reviewed-new")
	if e != nil {
		t.Fatal(e)
	}
	push, e := PreparePushForNewBranch(root, commit.NewBranch)
	if e != nil {
		t.Fatal(e)
	}
	creates := 0
	fake := func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		if args[1] == "list" {
			return []byte(`[]`), nil
		}
		creates++
		for _, arg := range args {
			if arg == "--draft" {
				t.Fatal("ready PR became draft")
			}
		}
		return []byte("https://github.com/example/project/pull/9"), nil
	}
	pr, e := preparePullRequestBranch(ctx, root, push, "Feature", "Reviewed body", false, commit.NewBranch, fake)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = PrepareStack(ctx, root, &commit, "Feature", &push, &pr); e != nil {
		t.Fatal(e)
	}
	if _, e = createPullRequestReviewed(ctx, root, pr, "", fake); e == nil {
		t.Fatal("PR accepted before new branch existed")
	}
	result, e := CommitSelectedReviewed(ctx, root, "Feature", commit)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = createPullRequestReviewed(ctx, root, pr, result.CommitSHA, fake); e == nil {
		t.Fatal("PR accepted before reviewed push")
	}
	// Simulate the acknowledged push in this synthetic fixture. No network call.
	run("config", "branch."+push.Branch+".remote", push.Remote)
	run("config", "branch."+push.Branch+".merge", "refs/heads/"+push.Target)
	run("update-ref", "refs/remotes/"+push.Remote+"/"+push.Target, result.CommitSHA)
	created, e := createPullRequestReviewed(ctx, root, pr, result.CommitSHA, fake)
	if e != nil || created.URL == "" || creates != 1 {
		t.Fatal(created, creates, e)
	}
}
