package git

import (
	"context"
	"errors"
	"testing"
)

func TestGitStackPreservesCompletedStepsAndStops(t *testing.T) {
	for _, failed := range []string{"commit", "push", "pr", ""} {
		t.Run(failed, func(t *testing.T) {
			var calls []string
			steps := stackSteps{commit: func() (SelectedCommitResult, error) {
				calls = append(calls, "commit")
				if failed == "commit" {
					return SelectedCommitResult{}, errors.New("hook failed")
				}
				return SelectedCommitResult{Committed: true, CommitSHA: "fixture"}, nil
			}, push: func(string) (string, error) {
				calls = append(calls, "push")
				if failed == "push" {
					return "", errors.New("push failed")
				}
				return "pushed", nil
			}, pr: func(string) (PullRequestResult, error) {
				calls = append(calls, "pr")
				if failed == "pr" {
					return PullRequestResult{}, errors.New("PR failed")
				}
				return PullRequestResult{URL: "https://github.com/example/project/pull/1"}, nil
			}}
			result, e := runStackSteps(context.Background(), "", steps)
			switch failed {
			case "commit":
				if result.Committed || len(calls) != 1 || e == nil {
					t.Fatal(result, calls, e)
				}
			case "push":
				if !result.Committed || result.Pushed || len(calls) != 2 || result.NextAction() != "push" {
					t.Fatal(result, calls, e)
				}
			case "pr":
				if !result.Committed || !result.Pushed || len(calls) != 3 || result.NextAction() != "create_pr" {
					t.Fatal(result, calls, e)
				}
			default:
				if e != nil || result.NextAction() != "open_pr" {
					t.Fatal(result, e)
				}
			}
		})
	}
}
func TestGitStackRetryOnlyRemainingStep(t *testing.T) {
	calls := 0
	result, e := runStackSteps(context.Background(), "already-committed", stackSteps{push: func(head string) (string, error) {
		calls++
		if head != "already-committed" {
			t.Fatal(head)
		}
		return "pushed", nil
	}})
	if e != nil || calls != 1 || result.Committed || !result.Pushed {
		t.Fatal(result, calls, e)
	}
}
func TestGitStackDefaultBranchHasNoPRFollowup(t *testing.T) {
	if got := (StackResult{Committed: true, Pushed: true, DefaultBranch: true}).NextAction(); got != "" {
		t.Fatal(got)
	}
}
func TestGitStackRechecksDestinationBeforeCommit(t *testing.T) {
	root, run, write := selectionRepo(t)
	run("switch", "-c", "feature/review")
	run("remote", "add", "origin", "https://github.com/example/project.git")
	write("a.txt", "changed\n")
	ctx := context.Background()
	commit, e := PrepareSelectedCommit(ctx, root, []string{"a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	push, e := PreparePush(root)
	if e != nil {
		t.Fatal(e)
	}
	plan, e := PrepareStack(ctx, root, &commit, "Reviewed commit", &push, nil)
	if e != nil {
		t.Fatal(e)
	}
	before := run("rev-parse", "HEAD")
	run("remote", "set-url", "origin", "https://github.com/another/project.git")
	if r, e := RunCommitStack(ctx, root, plan); e == nil || r.Committed {
		t.Fatal(r, e)
	}
	if after := run("rev-parse", "HEAD"); before != after {
		t.Fatal("commit ran before destination validation")
	}
}
func TestGitStackAllowsReviewedFirstCommit(t *testing.T) {
	root, run, _ := selectionRepo(t)
	run("symbolic-ref", "HEAD", "refs/heads/unborn-fixture")
	ctx := context.Background()
	commit, e := PrepareSelectedCommit(ctx, root, []string{"a.txt"}, "")
	if e != nil {
		t.Fatal(e)
	}
	plan, e := PrepareStack(ctx, root, &commit, "First commit", nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	result, e := RunCommitStack(ctx, root, plan)
	if e != nil || !result.Committed || result.CommitSHA == "" {
		t.Fatal(result, e)
	}
}
