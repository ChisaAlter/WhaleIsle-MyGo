// Package git reads lightweight workspace Git state for M7 chrome (branch, dirty)
// and runs fetch/pull/push via the git CLI.
package git

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotRepo means workspaceRoot is empty or not inside a Git work tree.
var ErrNotRepo = errors.New("git: not a repository")

// ErrEmptyCommitMessage means CommitAll was called with a blank message.
var ErrEmptyCommitMessage = errors.New("git: empty commit message")

// DefaultCommandTimeout bounds fetch/pull/push when no deadline is passed.
const DefaultCommandTimeout = 2 * time.Minute

// Snapshot is branch name plus whether the work tree has local changes.
type Snapshot struct {
	Branch string
	Dirty  bool
}

type runner func(dir string, name string, args ...string) ([]byte, error)

type combinedRunner func(ctx context.Context, dir, name string, args ...string) ([]byte, error)

func execRunner(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Output()
}

func combinedExecRunner(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return observedCombinedOutput(ctx, cmd)
}

func cleanWorkspaceRoot(workspaceRoot string) string {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return ""
	}
	return filepath.Clean(root)
}

// WorkspaceSnapshot runs `git branch --show-current` and `git status -sb` in workspaceRoot.
func WorkspaceSnapshot(workspaceRoot string) (Snapshot, error) {
	return workspaceSnapshot(workspaceRoot, execRunner)
}

func workspaceSnapshot(workspaceRoot string, run runner) (Snapshot, error) {
	root := cleanWorkspaceRoot(workspaceRoot)
	if root == "" {
		return Snapshot{}, ErrNotRepo
	}

	branchOut, err := run(root, "git", "branch", "--show-current")
	if err != nil {
		return Snapshot{}, ErrNotRepo
	}
	branch := strings.TrimSpace(string(branchOut))
	if branch == "" {
		branch = "HEAD"
	}

	statusOut, err := run(root, "git", "status", "-sb")
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Branch: branch,
		Dirty:  dirtyFromShortStatus(statusOut),
	}, nil
}

func dirtyFromShortStatus(out []byte) bool {
	lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
	if len(lines) <= 1 {
		return false
	}
	for _, ln := range lines[1:] {
		if len(bytes.TrimSpace(ln)) > 0 {
			return true
		}
	}
	return false
}

// FormatBadge returns a compact title-bar label (branch plus dirty marker).
func FormatBadge(s Snapshot) string {
	if s.Branch == "" {
		return ""
	}
	if s.Dirty {
		return s.Branch + " •"
	}
	return s.Branch
}

// Fetch runs `git fetch` in workspaceRoot and returns combined stdout/stderr.
func Fetch(workspaceRoot string) (string, error) {
	return runGitCommand(workspaceRoot, DefaultCommandTimeout, combinedExecRunner, "fetch")
}

// Pull updates only by fast-forward; divergent history is left for explicit resolution.
func Pull(workspaceRoot string) (string, error) {
	return runGitCommand(workspaceRoot, DefaultCommandTimeout, combinedExecRunner, "pull", "--ff-only")
}

// Push runs `git push` in workspaceRoot and returns combined stdout/stderr.
func Push(workspaceRoot string) (string, error) {
	return runGitCommand(workspaceRoot, DefaultCommandTimeout, combinedExecRunner, "push")
}

// CommitAll runs `git add -A` then `git commit -m` in workspaceRoot.
func CommitAll(workspaceRoot, message string) (string, error) {
	return commitAll(workspaceRoot, message, DefaultCommandTimeout, combinedExecRunner)
}

func commitAll(workspaceRoot, message string, timeout time.Duration, run combinedRunner) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", ErrEmptyCommitMessage
	}
	root := cleanWorkspaceRoot(workspaceRoot)
	if root == "" {
		return "", ErrNotRepo
	}
	if _, err := runGitCommand(root, timeout, run, "add", "-A"); err != nil {
		return "", err
	}
	return runGitCommand(root, timeout, run, "commit", "-m", message)
}

// Diff runs `git diff` in workspaceRoot, optionally scoped to rel paths under the repo root.
// Git exits 1 when there are differences; that case is treated as success with output.
func Diff(workspaceRoot string, paths ...string) (string, error) {
	return runGitDiff(workspaceRoot, DefaultCommandTimeout, combinedDiffRunner, paths...)
}

func combinedDiffRunner(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return out, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return out, nil
	}
	return out, err
}

func runGitDiff(workspaceRoot string, timeout time.Duration, run combinedRunner, paths ...string) (string, error) {
	root := cleanWorkspaceRoot(workspaceRoot)
	if root == "" {
		return "", ErrNotRepo
	}
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	ctxCheck, cancelCheck := context.WithTimeout(context.Background(), timeout)
	defer cancelCheck()
	if _, err := run(ctxCheck, root, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", ErrNotRepo
	}
	args := []string{"diff"}
	if trimmed := trimPathArgs(paths); len(trimmed) > 0 {
		args = append(args, "--")
		args = append(args, trimmed...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := run(ctx, root, "git", args...)
	return strings.TrimSpace(string(out)), err
}

func trimPathArgs(paths []string) []string {
	var out []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func runGitCommand(workspaceRoot string, timeout time.Duration, run combinedRunner, subcommand string, extraArgs ...string) (string, error) {
	root := cleanWorkspaceRoot(workspaceRoot)
	if root == "" {
		return "", ErrNotRepo
	}
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	args := gitCLIArgs(subcommand, extraArgs...)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := run(ctx, root, "git", args...)
	return strings.TrimSpace(string(out)), err
}

// gitCLIArgs builds argv passed to the git executable (subcommand first).
func gitCLIArgs(subcommand string, extra ...string) []string {
	args := make([]string, 0, 1+len(extra))
	args = append(args, subcommand)
	args = append(args, extra...)
	return args
}
