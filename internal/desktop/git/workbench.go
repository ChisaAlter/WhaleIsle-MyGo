package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Change describes both index and working-tree state. Paths are workspace-relative.
type Change struct {
	Path, Original  string
	Index, Worktree byte
}

func (c Change) Staged() bool   { return c.Index != ' ' && c.Index != '?' }
func (c Change) Unstaged() bool { return c.Worktree != ' ' || c.Index == '?' }

func localCommand(root, command string, args ...string) (string, error) {
	out, err := runGitCommand(root, 10*time.Second, combinedExecRunner, command, args...)
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", command, err, out)
	}
	return out, nil
}

func Changes(root string) ([]Change, error) {
	prefix, err := localCommand(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	return changesRaw(root, prefix)
}

func changesRaw(root, prefix string) ([]Change, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := combinedExecRunner(ctx, root, "git", "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ".")
	if err != nil {
		return nil, fmt.Errorf("git status: %w: %s", err, out)
	}
	rows := strings.Split(string(out), "\x00")
	changes := []Change{}
	for i := 0; i < len(rows); i++ {
		r := rows[i]
		if len(r) < 4 {
			continue
		}
		c := Change{Index: r[0], Worktree: r[1], Path: strings.TrimPrefix(r[3:], prefix)}
		if c.Index == 'R' || c.Index == 'C' || c.Worktree == 'R' || c.Worktree == 'C' {
			i++
			if i < len(rows) {
				c.Original = strings.TrimPrefix(rows[i], prefix)
			}
		}
		changes = append(changes, c)
	}
	return changes, nil
}

func literalPath(path string) error {
	if path == "" || filepath.IsAbs(path) || path == "." || strings.ContainsRune(path, 0) {
		return errors.New("select one workspace-relative file")
	}
	p := filepath.Clean(path)
	if p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
		return errors.New("path is outside the workspace")
	}
	return nil
}

// StagePaths stages only the explicitly supplied file paths; never the whole tree.
func StagePaths(root string, paths ...string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("no files selected")
	}
	for _, p := range paths {
		if err := literalPath(p); err != nil {
			return "", err
		}
	}
	return localCommand(root, "--literal-pathspecs", append([]string{"add", "--"}, paths...)...)
}
func UnstagePaths(root string, paths ...string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("no files selected")
	}
	for _, p := range paths {
		if err := literalPath(p); err != nil {
			return "", err
		}
	}
	if _, err := localCommand(root, "rev-parse", "--verify", "HEAD"); err != nil {
		return localCommand(root, "--literal-pathspecs", append([]string{"rm", "--cached", "--"}, paths...)...)
	}
	return localCommand(root, "--literal-pathspecs", append([]string{"reset", "HEAD", "--"}, paths...)...)
}
func IndexReview(root string) (text, token string, err error) {
	text, err = localCommand(root, "diff", "--cached", "--binary", "--no-ext-diff", "--no-textconv")
	if err != nil {
		return
	}
	if text == "" {
		err = errors.New("no staged changes; stage files first")
		return
	}
	sum := sha256.Sum256([]byte(text))
	token = hex.EncodeToString(sum[:])
	return
}

// CommitReviewed refuses a changed index after the user reviewed its exact diff.
func CommitReviewed(root, message, token string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", ErrEmptyCommitMessage
	}
	_, now, err := IndexReview(root)
	if err != nil {
		return "", err
	}
	if token == "" || now != token {
		return "", errors.New("staged changes changed; review again before committing")
	}
	return localCommand(root, "commit", "-m", message)
}

// CommitStaged never adds files and leaves unstaged changes untouched.
func CommitStaged(root, message string) (string, error) {
	_, token, err := IndexReview(root)
	if err != nil {
		return "", err
	}
	return CommitReviewed(root, message, token)
}
func DiffScope(root, path string, staged bool) (string, error) {
	args := []string{"diff", "--no-ext-diff", "--no-textconv"}
	if staged {
		args = append(args, "--cached")
	}
	if path != "" {
		if err := literalPath(path); err != nil {
			return "", err
		}
		args = append(args, "--", path)
	}
	out, err := localCommand(root, "--literal-pathspecs", args...)
	if err != nil || out != "" || staged || path == "" {
		return out, err
	}
	changes, err := Changes(root)
	if err != nil {
		return "", err
	}
	for _, c := range changes {
		if c.Path == path && c.Index == '?' {
			p := filepath.Join(root, filepath.FromSlash(path))
			info, e := os.Lstat(p)
			if e != nil {
				return "", e
			}
			if !info.Mode().IsRegular() || info.Size() > 512*1024 {
				return "", errors.New("untracked file is binary, a link, or too large to preview")
			}
			data, e := os.ReadFile(p)
			if e != nil {
				return "", e
			}
			if strings.ContainsRune(string(data), 0) {
				return "", errors.New("binary file cannot be previewed")
			}
			lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
			return fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 100644\n--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n+%s", path, path, path, len(lines), strings.Join(lines, "\n+")), nil
		}
	}
	return "", nil
}

// StagedSummary lists the exact repository-wide paths included by a staged commit.
func StagedSummary(root string) (string, error) {
	return localCommand(root, "diff", "--cached", "--name-status", "--no-ext-diff", "--no-textconv")
}
