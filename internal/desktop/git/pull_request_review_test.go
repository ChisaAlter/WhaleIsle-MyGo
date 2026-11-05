package git

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestGitPullRequestReviewBindsTargetAndPayload(t *testing.T) {
	root, run, _ := selectionRepo(t)
	run("switch", "-c", "feature/review")
	run("remote", "add", "origin", "https://github.com/example/project.git")
	push, e := PreparePush(root)
	if e != nil {
		t.Fatal(e)
	}
	fake := func(_ context.Context, _, name string, args ...string) ([]byte, error) {
		if name != "gh" {
			t.Fatal(name)
		}
		if strings.Join(args, " ") != "repo view --json nameWithOwner,url,defaultBranchRef" {
			t.Fatal(args)
		}
		return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
	}
	p, e := preparePullRequest(context.Background(), root, push, "Reviewed title", "Reviewed body", fake)
	if e != nil || p.Repository != "example/project" || p.Base != "main" {
		t.Fatal(p, e)
	}
	p.Title = "changed"
	if _, e = createPullRequestReviewed(context.Background(), root, p, "", fake); e == nil {
		t.Fatal("mutated PR payload accepted")
	}
}
func TestGitPullRequestExistingIsReused(t *testing.T) {
	root, run, _ := selectionRepo(t)
	run("switch", "-c", "feature/review")
	run("remote", "add", "origin", "git@github.com:example/project.git")
	push, _ := PreparePush(root)
	calls := 0
	fake := func(_ context.Context, _, name string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		calls++
		if args[0] != "pr" || args[1] != "list" {
			t.Fatal("would create duplicate", args)
		}
		return []byte(`[{"number":4,"title":"Existing","state":"OPEN","headRefName":"feature/review","baseRefName":"main","headRepository":{"name":"project"},"headRepositoryOwner":{"login":"example"},"url":"https://github.com/example/project/pull/4"}]`), nil
	}
	p, e := preparePullRequest(context.Background(), root, push, "Title", "Body", fake)
	if e != nil {
		t.Fatal(e)
	}
	result, e := createPullRequestReviewed(context.Background(), root, p, "", fake)
	if e != nil || !result.Existing || calls != 1 {
		t.Fatal(result, calls, e)
	}
}
func TestGitPullRequestRejectsForeignURL(t *testing.T) {
	for _, url := range []string{"https://github.com/other/project/pull/1", "https://evil.invalid/example/project/pull/1", "https://github.com/example/project/pull/1?token=x"} {
		if ValidPullRequestURL(url, "example/project") {
			t.Fatal(url)
		}
	}
}

func prFixture(t *testing.T) (string, func(...string) string, PushPlan) {
	t.Helper()
	root, run, _ := selectionRepo(t)
	run("switch", "-c", "feature/review")
	run("remote", "add", "origin", "https://github.com/example/project.git")
	p, e := PreparePush(root)
	if e != nil {
		t.Fatal(e)
	}
	return root, run, p
}
func TestGitPullRequestLookupFailureStopsCreation(t *testing.T) {
	root, _, push := prFixture(t)
	calls := 0
	fake := func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		calls++
		if args[1] != "list" {
			t.Fatal(args)
		}
		return nil, errors.New("unavailable")
	}
	p, e := preparePullRequest(context.Background(), root, push, "Title", "Body", fake)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = createPullRequestReviewed(context.Background(), root, p, "", fake); e == nil || calls != 1 {
		t.Fatal(calls, e)
	}
}
func TestGitPullRequestRequiresExactPushedHead(t *testing.T) {
	root, _, push := prFixture(t)
	fake := func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		if args[1] != "list" {
			t.Fatal("implicit push/create", args)
		}
		return []byte(`[]`), nil
	}
	p, e := preparePullRequest(context.Background(), root, push, "Title", "Body", fake)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = createPullRequestReviewed(context.Background(), root, p, "", fake); e == nil || !strings.Contains(e.Error(), "push this exact") {
		t.Fatal(e)
	}
}
func TestGitPullRequestCreationUsesReviewedDraftAndDoesNotBlindRetry(t *testing.T) {
	root, run, push := prFixture(t)
	run("config", "branch.feature/review.remote", "origin")
	run("config", "branch.feature/review.merge", "refs/heads/feature/review")
	run("update-ref", "refs/remotes/origin/feature/review", "HEAD")
	creates, lists := 0, 0
	var bodyFile string
	fake := func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		if args[1] == "list" {
			lists++
			return []byte(`[]`), nil
		}
		creates++
		want := []string{"pr", "create", "--draft", "--repo", "example/project", "--base", "main", "--head", "feature/review", "--title", "Reviewed title", "--body-file"}
		if len(args) != len(want)+1 {
			t.Fatal(args)
		}
		for i, v := range want {
			if args[i] != v {
				t.Fatal(args)
			}
		}
		bodyFile = args[len(args)-1]
		body, e := os.ReadFile(bodyFile)
		if e != nil || string(body) != "Reviewed body" {
			t.Fatal(string(body), e)
		}
		return nil, errors.New("connection lost after send")
	}
	p, e := preparePullRequest(context.Background(), root, push, "Reviewed title", "Reviewed body", fake)
	if e != nil {
		t.Fatal(e)
	}
	r, e := createPullRequestReviewed(context.Background(), root, p, "", fake)
	if e == nil || !r.Uncertain || creates != 1 || lists != 2 {
		t.Fatal(r, e, creates, lists)
	}
	if _, e := os.Stat(bodyFile); !os.IsNotExist(e) {
		t.Fatal("body temp file retained", e)
	}
}
func TestGitPullRequestRejectsStaleHeadAndForeignFork(t *testing.T) {
	root, run, push := prFixture(t)
	lists := 0
	fake := func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		lists++
		return []byte(`[{"number":1,"state":"OPEN","url":"https://github.com/example/project/pull/1","headRefName":"feature/review","baseRefName":"main","headRepository":{"name":"project"},"headRepositoryOwner":{"login":"another"}}]`), nil
	}
	p, e := preparePullRequest(context.Background(), root, push, "Title", "Body", fake)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = createPullRequestReviewed(context.Background(), root, p, "", fake); e == nil {
		t.Fatal("foreign fork accepted")
	}
	run("commit", "--allow-empty", "-m", "new head")
	if _, e = createPullRequestReviewed(context.Background(), root, p, "", fake); e == nil || lists != 1 {
		t.Fatal("stale head accepted", e, lists)
	}
}
