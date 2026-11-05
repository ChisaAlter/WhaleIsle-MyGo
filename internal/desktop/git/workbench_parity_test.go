package git

import (
	"context"
	"strings"
	"testing"
)

func TestPRSelectedUpstreamForkEnterpriseAndMergeBase(t *testing.T) {
	root, run, _ := selectionRepo(t)
	run("switch", "-c", "feature/review")
	run("remote", "add", "origin", "git@ghe.example.test:contributor/fork.git")
	run("config", "branch.feature/review.gh-merge-base", "release")
	push, err := PreparePush(root)
	if err != nil {
		t.Fatal(err)
	}
	fake := func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			if strings.Join(args, " ") != "repo view --json nameWithOwner,url,defaultBranchRef" {
				t.Fatal(args)
			}
			return []byte(`{"nameWithOwner":"upstream/project","url":"https://ghe.example.test/upstream/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--repo ghe.example.test/upstream/project --head contributor:feature/review --base release") {
			t.Fatal(args)
		}
		return []byte(`[{"number":3,"state":"OPEN","url":"https://ghe.example.test/upstream/project/pull/3","headRefName":"feature/review","baseRefName":"release","headRepository":{"name":"fork"},"headRepositoryOwner":{"login":"contributor"}}]`), nil
	}
	p, err := preparePullRequest(context.Background(), root, push, "Reviewed", "Body", fake)
	if err != nil || p.Host != "ghe.example.test" || p.Repository != "upstream/project" || p.HeadRepository != "contributor/fork" || p.Head != "contributor:feature/review" || p.Base != "release" {
		t.Fatal(p, err)
	}
	r, err := lookupPullRequest(context.Background(), root, p, fake)
	if err != nil || !r.Existing || !ValidPullRequestURLForHost(r.URL, r.Host, r.Repository) {
		t.Fatal(r, err)
	}
	p.Head = "attacker:branch"
	if err := validatePullRequest(context.Background(), root, p, ""); err == nil {
		t.Fatal("edited fork target accepted")
	}
}

func TestPRSameBranchNameAcrossForkIsValid(t *testing.T) {
	root, run, _ := selectionRepo(t)
	run("remote", "add", "origin", "https://github.com/contributor/fork.git")
	push, err := PreparePush(root)
	if err != nil {
		t.Fatal(err)
	}
	fake := func(context.Context, string, string, ...string) ([]byte, error) {
		return []byte(`{"nameWithOwner":"upstream/project","url":"https://github.com/upstream/project","defaultBranchRef":{"name":"main"}}`), nil
	}
	p, err := preparePullRequest(context.Background(), root, push, "Title", "Body", fake)
	if err != nil || p.Head != "contributor:main" || p.Base != "main" {
		t.Fatal(p, err)
	}
}

func TestRepositoryURLAndProgressDetails(t *testing.T) {
	for raw, want := range map[string]string{"git@ghe.example.test:team/project.git": "https://ghe.example.test/team/project", "https://github.com/team/project": "https://github.com/team/project"} {
		if got, err := RepositoryWebURL(raw); err != nil || got != want {
			t.Fatal(raw, got, err)
		}
	}
	for _, raw := range []string{"https://token@github.com/team/project", "https://github.com/team/project?token=x", "https://github.com/team/project/pull/1"} {
		if _, err := RepositoryWebURL(raw); err == nil {
			t.Fatal(raw)
		}
	}
	p := NewProgress("fixture", "Failed", nil)
	full := strings.Repeat("full command output\n", 40) + "final error detail"
	p.SetDetails(full)
	p.SetRepositoryAction("https://ghe.example.test/team/project")
	p.Finish("failed", true)
	s := p.Snapshot()
	if s.Details != full || !s.Failed || s.ActionURL != "https://ghe.example.test/team/project" {
		t.Fatal(s)
	}
}
