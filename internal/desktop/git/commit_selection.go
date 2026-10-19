package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const LargeFileLimit int64 = 100 * 1024 * 1024

type LargeFile struct {
	Path string
	Size int64
}

// SelectedCommitPlan binds the reviewed tree, original index, HEAD and repository.
// Selected files include their current saved working copy, even if partially staged.
// The private token is never accepted from a remote/UI-supplied arbitrary hash.
type SelectedCommitPlan struct {
	Paths                                               []string
	Summary, Patch, Branch, NewBranch                   string
	LargeFiles                                          []LargeFile
	root, gitDir, indexPath, token, tree, head, headRef string
}
type SelectedCommitResult struct {
	Output, CommitSHA, CreatedBranch string
	Committed                        bool
}

func selectedCommand(ctx context.Context, root, index string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	for _, e := range gitChildEnv() {
		key := strings.SplitN(e, "=", 2)[0]
		if strings.HasPrefix(key, "GIT_") && (key == "GIT_INDEX_FILE" || key == "GIT_DIR" || key == "GIT_WORK_TREE" || key == "GIT_COMMON_DIR" || key == "GIT_OPTIONAL_LOCKS") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	cmd.Env = append(cmd.Env, "GIT_OPTIONAL_LOCKS=0")
	if index != "" {
		cmd.Env = append(cmd.Env, "GIT_INDEX_FILE="+index)
	}
	var out []byte
	var err error
	if shouldObserveGit(args) {
		out, err = observedCombinedOutput(ctx, cmd)
	} else {
		out, err = cmd.CombinedOutput()
	}
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return out, nil
}
func selectedText(ctx context.Context, root, index string, args ...string) (string, error) {
	out, e := selectedCommand(ctx, root, index, args...)
	return strings.TrimSpace(string(out)), e
}
func selectedChanges(ctx context.Context, root string) ([]Change, error) {
	prefix, err := selectedText(ctx, root, "", "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	// Do not path-limit status: it would turn a cross-workspace rename into an
	// apparent addition and lose the source path that must be reviewed together.
	out, err := selectedCommand(ctx, root, "", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	rows := strings.Split(string(out), "\x00")
	var result []Change
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 4 {
			continue
		}
		c := Change{Index: row[0], Worktree: row[1], Path: row[3:]}
		if strings.ContainsAny(string([]byte{c.Index, c.Worktree}), "RC") {
			i++
			if i < len(rows) {
				c.Original = rows[i]
			}
		}
		if prefix != "" {
			inside := strings.HasPrefix(c.Path, prefix)
			sourceInside := c.Original != "" && strings.HasPrefix(c.Original, prefix)
			if c.Original != "" && inside != sourceInside {
				return nil, errors.New("rename crosses the workspace boundary; open the repository root to review it")
			}
			if !inside {
				continue
			}
		}
		c.Path = strings.TrimPrefix(c.Path, prefix)
		c.Original = strings.TrimPrefix(c.Original, prefix)
		result = append(result, c)
	}
	return result, nil
}

func checkLargePaths(root string, changes []Change) ([]LargeFile, error) {
	var files []LargeFile
	for _, c := range changes {
		if err := literalPath(c.Path); err != nil {
			return nil, err
		}
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(c.Path)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() && info.Size() > LargeFileLimit {
			files = append(files, LargeFile{Path: c.Path, Size: info.Size()})
		}
	}
	return files, nil
}

// CheckLargeFiles is read-only and reports strict >100 MiB, never a silent filter.
func CheckLargeFiles(root string) ([]LargeFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	changes, e := selectedChanges(ctx, root)
	if e != nil {
		return nil, e
	}
	return checkLargePaths(root, changes)
}
func temporaryIndex(gitDir string) (string, error) {
	f, e := os.CreateTemp(gitDir, "mygo-commit-index-*")
	if e != nil {
		return "", e
	}
	name := f.Name()
	f.Close()
	if e = os.Remove(name); e != nil {
		return "", e
	}
	return name, nil
}
func removeIndex(index string) { _ = os.Remove(index); _ = os.Remove(index + ".lock") }

func PrepareSelectedCommit(ctx context.Context, root string, paths []string, newBranch string) (SelectedCommitPlan, error) {
	p, index, e := prepareSelectedCommit(ctx, root, paths, newBranch)
	if index != "" {
		defer removeIndex(index)
	}
	return p, e
}
func prepareSelectedCommit(ctx context.Context, root string, paths []string, newBranch string) (p SelectedCommitPlan, index string, err error) {
	if cleanWorkspaceRoot(root) == "" {
		return p, "", ErrNotRepo
	}
	if len(paths) == 0 {
		return p, "", errors.New("select at least one file to commit")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return
	}
	p.root = root
	p.gitDir, err = selectedText(ctx, root, "", "rev-parse", "--absolute-git-dir")
	if err != nil {
		return
	}
	p.indexPath, err = selectedText(ctx, root, "", "rev-parse", "--path-format=absolute", "--git-path", "index")
	if err != nil {
		return
	}
	if value, e := selectedText(ctx, root, "", "rev-parse", "--verify", "HEAD"); e == nil {
		p.head = value
	}
	if value, e := selectedText(ctx, root, "", "symbolic-ref", "-q", "HEAD"); e == nil {
		p.headRef = value
	}
	p.Branch = strings.TrimPrefix(p.headRef, "refs/heads/")
	if p.Branch == "" {
		p.Branch = "HEAD (detached)"
	}
	newBranch = strings.TrimSpace(newBranch)
	if newBranch != "" {
		if e := checkBranchName(root, newBranch); e != nil {
			err = e
			return
		}
		if _, e := selectedText(ctx, root, "", "show-ref", "--verify", "--quiet", "refs/heads/"+newBranch); e == nil {
			err = errors.New("branch already exists; choose a new name")
			return
		}
		p.NewBranch = newBranch
	}
	if e := requireOrdinaryCommitState(ctx, root); e != nil {
		err = e
		return
	}
	changes, e := selectedChanges(ctx, root)
	if e != nil {
		err = e
		return
	}
	available := map[string]Change{}
	for _, c := range changes {
		available[c.Path] = c
	}
	seen := map[string]bool{}
	var chosen []Change
	for _, path := range paths {
		c, ok := available[path]
		if !ok {
			err = fmt.Errorf("%s has no changes; refresh the selection", path)
			return
		}
		if strings.ContainsAny(string([]byte{c.Index, c.Worktree}), "U") || string([]byte{c.Index, c.Worktree}) == "AA" || string([]byte{c.Index, c.Worktree}) == "DD" {
			err = errors.New("resolve merge conflicts before committing")
			return
		}
		if e := selectedPath(root, path); e != nil {
			err = e
			return
		}
		chosen = append(chosen, c)
		for _, item := range []string{path, c.Original} {
			if item != "" && !seen[item] {
				if e := literalPath(item); e != nil {
					err = e
					return
				}
				seen[item] = true
				p.Paths = append(p.Paths, item)
			}
		}
	}
	sort.Strings(p.Paths)
	p.LargeFiles, err = checkLargePaths(root, chosen)
	if err != nil {
		return
	}
	index, err = temporaryIndex(p.gitDir)
	if err != nil {
		return
	}
	scratchHooks, hooksErr := os.MkdirTemp(p.gitDir, "mygo-index-hooks-")
	if hooksErr != nil {
		err = hooksErr
		return
	}
	defer os.RemoveAll(scratchHooks)
	scratch := func(args ...string) (string, error) {
		return selectedText(ctx, root, index, append([]string{"-c", "core.hooksPath=" + filepath.ToSlash(scratchHooks)}, args...)...)
	}
	base := p.head
	if base == "" {
		base = "--empty"
	}
	if _, err = scratch("read-tree", base); err != nil {
		return
	}
	args := append([]string{"--literal-pathspecs", "add", "-A", "--"}, p.Paths...)
	if _, err = scratch(args...); err != nil {
		return
	}
	p.tree, err = scratch("write-tree")
	if err != nil {
		return
	}
	p.Summary, err = scratch("diff", "--cached", "--name-status", "--no-ext-diff", "--no-textconv")
	if err != nil {
		return
	}
	if p.Summary == "" {
		err = errors.New("selected files have no changes relative to HEAD")
		return
	}
	p.Patch, err = scratch("diff", "--cached", "--patch", "--no-ext-diff", "--no-textconv")
	if err != nil {
		return
	}
	original, e := os.ReadFile(p.indexPath)
	if e != nil && !os.IsNotExist(e) {
		err = e
		return
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00", p.root, p.gitDir, p.indexPath, p.head, p.headRef, p.NewBranch, p.tree, strings.Join(p.Paths, "\x00"))
	h.Write(original)
	p.token = hex.EncodeToString(h.Sum(nil))
	return
}

// CommitSelectedReviewed commits an isolated index, holding the real index lock.
// Only selected entries are reconciled after success. Unselected staging is kept
// byte-for-byte semantically; a failure never resets or unstages the real index.
func CommitSelectedReviewed(ctx context.Context, root, message string, review SelectedCommitPlan) (result SelectedCommitResult, err error) {
	if strings.TrimSpace(message) == "" {
		return result, ErrEmptyCommitMessage
	}
	if review.token == "" {
		return result, errors.New("review the selected files before committing")
	}
	canonical, e := filepath.Abs(root)
	if e != nil {
		return result, e
	}
	canonical, e = filepath.EvalSymlinks(canonical)
	if e != nil || canonical != review.root {
		return result, errors.New("repository changed; review again")
	}
	gitDir, e := selectedText(ctx, root, "", "rev-parse", "--absolute-git-dir")
	if e != nil || gitDir != review.gitDir {
		return result, errors.New("repository changed; review again")
	}
	lock, err := os.OpenFile(review.indexPath+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, fmt.Errorf("repository index is busy: %w", err)
	}
	lockPath := lock.Name()
	defer func() { lock.Close(); _ = os.Remove(lockPath) }()
	// Rebuild from the selected worktree under the lock; includes index and HEAD guards.
	current, index, e := prepareSelectedCommit(ctx, root, reviewPaths(review), review.NewBranch)
	if index != "" {
		defer removeIndex(index)
	}
	if e != nil {
		return result, e
	}
	if current.token != review.token {
		return result, errors.New("repository, index or selected files changed; review again before committing")
	}
	original, e := os.ReadFile(review.indexPath)
	if e != nil && !os.IsNotExist(e) {
		return result, e
	}
	scratchHooks, e := os.MkdirTemp(review.gitDir, "mygo-index-hooks-")
	if e != nil {
		return result, e
	}
	defer os.RemoveAll(scratchHooks)
	mergeIndex, e := temporaryIndex(review.gitDir)
	if e != nil {
		return result, e
	}
	defer removeIndex(mergeIndex)
	if len(original) > 0 {
		if e = os.WriteFile(mergeIndex, original, 0600); e != nil {
			return result, e
		}
	} else if _, e = selectedText(ctx, root, mergeIndex, "-c", "core.hooksPath="+filepath.ToSlash(scratchHooks), "read-tree", "--empty"); e != nil {
		return result, e
	}
	if review.NewBranch != "" {
		emitPhase(ctx, "Preparing feature branch…")
		if _, e = selectedText(ctx, root, index, "switch", "-c", review.NewBranch); e != nil {
			return result, e
		}
		result.CreatedBranch = review.NewBranch
	}
	emitPhase(ctx, "Committing selected files…")
	hooks, hookErr := guardedCommitHooks(ctx, root, review)
	if hookErr != nil {
		return result, hookErr
	}
	defer os.RemoveAll(hooks)
	out, e := selectedText(ctx, root, index, "-c", "core.hooksPath="+filepath.ToSlash(hooks), "commit", "-m", strings.TrimSpace(message))
	result.Output = out
	head, _ := selectedText(context.WithoutCancel(ctx), root, "", "rev-parse", "--verify", "HEAD")
	if head == review.head {
		if e != nil {
			return result, e
		}
		return result, errors.New("Git returned without creating a commit; refresh before retrying")
	}
	result.Committed = true
	result.CommitSHA = head
	latestIndex, indexReadErr := os.ReadFile(review.indexPath)
	if indexReadErr != nil && !os.IsNotExist(indexReadErr) {
		return result, indexReadErr
	}
	if !bytes.Equal(original, latestIndex) {
		return result, fmt.Errorf("commit %s was created, but the real index changed outside its lock; it was preserved. Inspect Git state before retrying", head)
	}
	// Even when a post-commit hook reports failure, reconcile the created commit.
	reconcileCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	args := append([]string{"-c", "core.hooksPath=" + filepath.ToSlash(scratchHooks), "--literal-pathspecs", "reset", head, "--"}, review.Paths...)
	if _, reconcileErr := selectedText(reconcileCtx, root, mergeIndex, args...); reconcileErr != nil {
		return result, fmt.Errorf("commit %s was created, but index reconciliation failed; do not repeat the commit: %w", head, reconcileErr)
	}
	merged, readErr := os.ReadFile(mergeIndex)
	if readErr != nil {
		return result, fmt.Errorf("commit %s was created; could not read preserved index: %w", head, readErr)
	}
	if _, writeErr := lock.Write(merged); writeErr != nil {
		return result, fmt.Errorf("commit %s was created; could not save preserved index: %w", head, writeErr)
	}
	if syncErr := lock.Sync(); syncErr != nil {
		return result, syncErr
	}
	if closeErr := lock.Close(); closeErr != nil {
		return result, closeErr
	}
	if renameErr := os.Rename(lockPath, review.indexPath); renameErr != nil {
		return result, fmt.Errorf("commit %s was created; index reconciliation failed: %w", head, renameErr)
	}
	return result, e
}
func reviewPaths(p SelectedCommitPlan) []string {
	// Rename sources are included in Paths for staging/deletion; only the status
	// destination is selectable. Re-read status to retain this distinction.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	changes, e := selectedChanges(ctx, p.root)
	if e != nil {
		return p.Paths
	}
	available := map[string]bool{}
	for _, c := range changes {
		available[c.Path] = true
	}
	var paths []string
	for _, path := range p.Paths {
		if available[path] {
			paths = append(paths, path)
		}
	}
	return paths
}

// Allow a tracked deletion whose parents were removed, while rejecting links,
// directories, metadata paths and pathspec syntax as filesystem traversal.
func selectedPath(root, path string) error {
	if err := literalPath(path); err != nil {
		return err
	}
	rel := filepath.FromSlash(path)
	if filepath.Clean(rel) != rel || strings.ContainsAny(path, "\\:") {
		return errors.New("select a literal workspace-relative file")
	}
	current := root
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.EqualFold(part, ".git") {
			return errors.New("Git metadata cannot be committed")
		}
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("linked paths require explicit Git handling")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return errors.New("invalid parent directory")
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return errors.New("select a regular file; directories and submodules require explicit Git handling")
		}
	}
	return nil
}
func requireOrdinaryCommitState(ctx context.Context, root string) error {
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "REBASE_HEAD", "rebase-merge", "rebase-apply", "sequencer"} {
		path, err := selectedText(ctx, root, "", "rev-parse", "--path-format=absolute", "--git-path", name)
		if err != nil {
			return err
		}
		if _, err = os.Stat(path); err == nil {
			return errors.New("finish or abort the current merge, rebase or sequencer operation before using selected commit")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	repo, err := selectedText(ctx, root, "", "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	unmerged, err := selectedText(ctx, repo, "", "ls-files", "--unmerged")
	if err != nil {
		return err
	}
	if unmerged != "" {
		return errors.New("resolve all unmerged index entries before committing")
	}
	return nil
}

func shellLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// Keep installed hooks, but fail closed if a pre-commit/message hook changes the
// reviewed tree or branch. A temporary hooksPath forwards every installed hook;
// it does not disable hooks or change repository configuration.
func guardedCommitHooks(ctx context.Context, root string, p SelectedCommitPlan) (string, error) {
	original, err := selectedText(ctx, root, "", "rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(p.gitDir, "mygo-commit-hooks-")
	if err != nil {
		return "", err
	}
	names := map[string]bool{"pre-commit": true, "prepare-commit-msg": true, "commit-msg": true, "reference-transaction": true}
	entries, err := os.ReadDir(original)
	if err != nil && !os.IsNotExist(err) {
		os.RemoveAll(dir)
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			names[entry.Name()] = true
		}
	}
	expectedRef := p.headRef
	if p.NewBranch != "" {
		expectedRef = "refs/heads/" + p.NewBranch
	}
	treeGuard := "tree=$(git write-tree) || exit 1\nif test \"$tree\" != " + shellLiteral(p.tree) + "; then\n echo 'Hook changed reviewed files. Review again before committing.' >&2\n exit 1\nfi\n"
	guard := treeGuard + "head=$(git rev-parse --verify HEAD 2>/dev/null || :)\n" +
		"ref=$(git symbolic-ref -q HEAD 2>/dev/null || :)\n" +
		"if test \"$head\" != " + shellLiteral(p.head) + " || test \"$ref\" != " + shellLiteral(expectedRef) + "; then\n" +
		" echo 'Hook changed HEAD. Review again before committing.' >&2\n exit 1\nfi\n"

	for name := range names {
		hook := filepath.ToSlash(filepath.Join(original, name))
		script := "#!/bin/sh\noriginal=" + shellLiteral(hook) + "\nif test -x \"$original\"; then\n \"$original\" \"$@\"\n code=$?\n if test \"$code\" != 0; then exit \"$code\"; fi\nfi\n"
		switch name {
		case "pre-commit", "prepare-commit-msg", "commit-msg":
			script += guard
		case "reference-transaction":
			script += "if test \"$1\" = prepared; then\n" + treeGuard + "fi\n"
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0700); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}
