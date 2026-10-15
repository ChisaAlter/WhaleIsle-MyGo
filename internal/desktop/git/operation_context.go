package git

import (
	"context"
	"errors"
)

func FetchContext(ctx context.Context, root string) (string, error) {
	if cleanWorkspaceRoot(root) == "" {
		return "", ErrNotRepo
	}
	emitPhase(ctx, "Fetching…")
	return selectedText(ctx, root, "", "fetch", "--progress")
}
func PullContext(ctx context.Context, root string) (string, error) {
	if cleanWorkspaceRoot(root) == "" {
		return "", ErrNotRepo
	}
	emitPhase(ctx, "Pulling (fast-forward only)…")
	return selectedText(ctx, root, "", "pull", "--ff-only", "--progress")
}
func PushReviewedContext(ctx context.Context, root string, reviewed PushPlan) (string, error) {
	current, e := PreparePush(root)
	if e != nil {
		return "", e
	}
	if current != reviewed {
		return "", errors.New("branch or push destination changed; review again")
	}
	emitPhase(ctx, "Pushing to "+current.Remote+"/"+current.Target+"…")
	return selectedText(ctx, root, "", "push", "--progress", "--set-upstream", current.Remote, "HEAD:refs/heads/"+current.Target)
}
