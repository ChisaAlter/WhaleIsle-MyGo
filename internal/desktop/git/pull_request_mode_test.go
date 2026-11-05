package git

import (
	"context"
	"strings"
	"testing"
)

func TestPullRequestModeIsReviewedAndPassedToGH(t *testing.T) {
	for _, draft := range []bool{false, true} {
		root, run, push := prFixture(t)
		run("config", "branch.feature/review.remote", "origin")
		run("config", "branch.feature/review.merge", "refs/heads/feature/review")
		run("update-ref", "refs/remotes/origin/feature/review", "HEAD")
		creates := 0
		fake := func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
			if args[0] == "repo" {
				return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
			}
			if args[1] == "list" {
				return []byte(`[]`), nil
			}
			creates++
			if strings.Contains(" "+strings.Join(args, " ")+" ", " --draft ") != draft {
				t.Fatalf("wrong mode: %v", args)
			}
			return []byte("https://github.com/example/project/pull/6"), nil
		}
		p, e := preparePullRequestMode(context.Background(), root, push, "Title", "Body", draft, fake)
		if e != nil {
			t.Fatal(e)
		}
		changed := p
		changed.Draft = !p.Draft
		if _, e = createPullRequestReviewed(context.Background(), root, changed, "", fake); e == nil {
			t.Fatal("changed mode did not invalidate review")
		}
		if creates != 0 {
			t.Fatal("changed mode sent PR")
		}
		result, e := createPullRequestReviewed(context.Background(), root, p, "", fake)
		if e != nil || result.URL == "" || creates != 1 {
			t.Fatalf("result: %+v %v", result, e)
		}
	}
}
