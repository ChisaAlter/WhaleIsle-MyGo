package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CompareBranch compares the merge-base of the selected branch and HEAD to HEAD.
// It never switches branches or includes staged, unstaged, or untracked changes.
func CompareBranch(ctx context.Context, root, path, baseRef string) (string, error) {
	if cleanWorkspaceRoot(root) == "" {
		return "", ErrNotRepo
	}
	if baseRef == "" || strings.TrimSpace(baseRef) != baseRef || strings.HasPrefix(baseRef, "-") || strings.ContainsAny(baseRef, "\x00\r\n\t") {
		return "", errors.New("select an existing branch")
	}
	if path != "" {
		if err := literalPath(path); err != nil {
			return "", err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	run := func(args ...string) (string, error) {
		out, err := combinedExecRunner(ctx, root, "git", args...)
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", fmt.Errorf("git comparison: %w: %s", err, out)
		}
		return strings.TrimSpace(string(out)), nil
	}
	listed, err := run("for-each-ref", "--format=%(refname)%09%(HEAD)%09%(symref)%09.", "refs/heads", "refs/remotes")
	if err != nil {
		return "", err
	}
	ref := ""
	for _, branch := range parseBranches(listed) {
		if branch.Name == baseRef {
			ref = "refs/heads/" + branch.Name
			if branch.Remote {
				ref = "refs/remotes/" + branch.Name
			}
			break
		}
	}
	if ref == "" {
		return "", errors.New("comparison branch is no longer available; refresh the branch list")
	}
	args := []string{"--literal-pathspecs", "diff", "--no-color", "--no-ext-diff", "--no-textconv", "--find-renames", ref + "...HEAD", "--"}
	if path != "" {
		args = append(args, path)
	}
	return run(args...)
}
