package git

import (
	"context"
	"errors"
	"strings"
)

// StackResult records completed stages even when a later action fails.
type StackResult struct {
	Output, CommitSHA, CreatedBranch string
	Committed, Pushed                bool
	PR                               PullRequestResult
	PRPlan                           *PullRequestPlan
	DefaultBranch                    bool
	PushPlan                         *PushPlan
}

func (r StackResult) NextAction() string {
	if r.PR.URL != "" {
		return "open_pr"
	}
	if r.PR.Uncertain {
		return "check_pr"
	}
	if r.Pushed {
		if !r.DefaultBranch {
			return "create_pr"
		}
		return ""
	}
	if r.Committed {
		return "push"
	}
	return ""
}

type stackSteps struct {
	commit func() (SelectedCommitResult, error)
	push   func(string) (string, error)
	pr     func(string) (PullRequestResult, error)
}

func runStackSteps(ctx context.Context, head string, steps stackSteps) (r StackResult, err error) {
	r.CommitSHA = head
	if e := ctx.Err(); e != nil {
		return r, e
	}
	if steps.commit != nil {
		v, e := steps.commit()
		r.Committed, r.CommitSHA, r.CreatedBranch = v.Committed, v.CommitSHA, v.CreatedBranch
		r.Output = v.Output
		if e != nil {
			return r, e
		}
	}
	if e := ctx.Err(); e != nil {
		return r, e
	}
	if steps.push != nil {
		out, e := steps.push(r.CommitSHA)
		r.Output = strings.TrimSpace(r.Output + "\n" + out)
		if e != nil {
			return r, e
		}
		r.Pushed = true
	}
	if e := ctx.Err(); e != nil {
		return r, e
	}
	if steps.pr != nil {
		v, e := steps.pr(r.CommitSHA)
		r.PR = v
		if e != nil {
			return r, e
		}
	}
	return r, nil
}

// StackPlan is immutable outside this package; retries omit completed stages.
type StackPlan struct {
	root, dir, head, message string
	commit                   *SelectedCommitPlan
	push                     *PushPlan
	pr                       *PullRequestPlan
}

func PrepareStack(ctx context.Context, root string, commit *SelectedCommitPlan, message string, push *PushPlan, pr *PullRequestPlan) (p StackPlan, err error) {
	p.root, p.dir, err = workflowIdentity(ctx, root)
	if err != nil {
		return
	}
	p.head, err = selectedText(ctx, root, "", "rev-parse", "HEAD")
	if err != nil {
		if commit == nil {
			return
		}
		p.head = ""
	}
	err = nil
	if commit == nil && push == nil && pr == nil {
		return p, errors.New("select an action to review")
	}
	if commit != nil {
		copy := *commit
		p.commit = &copy
		p.message = message
	}
	if push != nil {
		current, e := reviewedBranchPush(root, stackNewBranch(p.commit))
		if e != nil {
			return p, e
		}
		if current != *push {
			return p, errors.New("push destination changed; review again")
		}
		copy := *push
		p.push = &copy
	}
	if pr != nil {
		if e := validatePullRequestBeforeCommit(ctx, root, *pr, p.head, stackNewBranch(p.commit)); e != nil {
			return p, e
		}
		if push != nil && *push != pr.Push {
			return p, errors.New("PR and push destinations differ")
		}
		copy := *pr
		p.pr = &copy
	}
	return
}
func RunCommitStack(ctx context.Context, root string, p StackPlan) (StackResult, error) {
	abs, dir, e := workflowIdentity(ctx, root)
	if e != nil {
		return StackResult{}, e
	}
	if abs != p.root || dir != p.dir {
		return StackResult{}, errors.New("repository changed; review again")
	}
	head, he := selectedText(ctx, root, "", "rev-parse", "HEAD")
	if he != nil {
		head = ""
	}
	if head != p.head {
		return StackResult{}, errors.New("HEAD changed; review again")
	}
	if p.push != nil {
		current, e := reviewedBranchPush(root, stackNewBranch(p.commit))
		if e != nil {
			return StackResult{}, e
		}
		if current != *p.push {
			return StackResult{}, errors.New("push destination changed; review again before committing")
		}
	}
	steps := stackSteps{}
	if p.commit != nil {
		steps.commit = func() (SelectedCommitResult, error) { return CommitSelectedReviewed(ctx, root, p.message, *p.commit) }
	}
	if p.push != nil {
		steps.push = func(expected string) (string, error) {
			head, e := selectedText(ctx, root, "", "rev-parse", "HEAD")
			if e != nil {
				return "", e
			}
			if head != expected {
				return "", errors.New("HEAD changed after commit; review push again")
			}
			return PushReviewedContext(ctx, root, *p.push)
		}
	}
	if p.pr != nil {
		steps.pr = func(head string) (PullRequestResult, error) { return CreatePullRequestReviewed(ctx, root, *p.pr, head) }
	}
	r, e := runStackSteps(ctx, p.head, steps)
	r.PRPlan = p.pr
	r.PushPlan = p.push
	if p.pr != nil && r.PushPlan == nil {
		copy := p.pr.Push
		r.PushPlan = &copy
	}
	if p.push != nil {
		r.DefaultBranch = p.push.DefaultBranch
	}
	if r.Committed && e != nil {
		r.Output += "\nLocal commit created: " + r.CommitSHA + ". Do not repeat the commit."
	}
	if r.Pushed && e != nil {
		r.Output += "\nPush completed. Only the PR step remains."
	}
	return r, e
}

func stackNewBranch(commit *SelectedCommitPlan) string {
	if commit != nil {
		return commit.NewBranch
	}
	return ""
}
func reviewedBranchPush(root, newBranch string) (PushPlan, error) {
	if newBranch != "" {
		return PreparePushForNewBranch(root, newBranch)
	}
	return PreparePush(root)
}
