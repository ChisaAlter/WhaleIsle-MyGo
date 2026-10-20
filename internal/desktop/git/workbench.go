package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Change describes both index and working-tree state. Paths are workspace-relative.
// Added and Deleted are the line counts of the whole change against HEAD, so a
// commit list can show the same +N/-N a diff does.
type Change struct {
	Path, Original  string
	Index, Worktree byte
	Added, Deleted  int
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
	attachDiffstat(ctx, root, prefix, changes)
	return changes, nil
}

// attachDiffstat fills each change's line counts. One numstat pass against HEAD
// covers staged and unstaged edits together, which is what the commit list means;
// untracked files are counted from disk, and binary files report zero.
func attachDiffstat(ctx context.Context, root, prefix string, changes []Change) {
	if len(changes) == 0 {
		return
	}
	counts := map[string][2]int{}
	// stdout only: a combined runner would fold git warnings into the first
	// record and turn its added count into a parse failure.
	if out, err := selectedCommand(ctx, root, "", "diff", "--numstat", "-z", "--no-renames", "HEAD", "--", "."); err == nil {
		readNumstat(counts, out, prefix)
	}
	for i := range changes {
		c := &changes[i]
		if c.Index == '?' {
			c.Added, c.Deleted = countUntrackedLines(root, c.Path)
			continue
		}
		if n, ok := counts[c.Path]; ok {
			c.Added, c.Deleted = n[0], n[1]
		}
	}
}

// readNumstat parses `git diff --numstat -z` into path → {added, deleted}.
func readNumstat(counts map[string][2]int, out []byte, prefix string) {
	for _, record := range strings.Split(string(out), "\x00") {
		// The runners fold stderr into stdout, so a warning may precede the
		// record; the record itself is always the last line of the chunk.
		if cut := strings.LastIndexByte(record, '\n'); cut >= 0 {
			record = record[cut+1:]
		}
		parts := strings.SplitN(record, "\t", 3)
		if len(parts) != 3 || parts[2] == "" {
			continue
		}
		added, errAdded := strconv.Atoi(parts[0])
		deleted, errDeleted := strconv.Atoi(parts[1])
		if errAdded != nil || errDeleted != nil {
			continue
		}
		counts[strings.TrimPrefix(parts[2], prefix)] = [2]int{added, deleted}
	}
}

// countUntrackedLines counts a file HEAD does not know yet. A large or binary
// file reports zero instead of guessing.
func countUntrackedLines(root, path string) (int, int) {
	info, err := os.Stat(filepath.Join(root, path))
	if err != nil || info.IsDir() || info.Size() > 4<<20 {
		return 0, 0
	}
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil || len(data) == 0 {
		return 0, 0
	}
	head := data
	if len(head) > 8000 {
		head = head[:8000]
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return 0, 0
	}
	lines := bytes.Count(data, []byte("\n"))
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines, 0
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

// DiffScope shows index-only or working-tree changes. The whole working-tree
// view includes untracked files without adding them to the index.
func DiffScope(root, path string, staged bool) (string, error) {
	return diffWorkingTree(root, path, staged, false)
}

// DiffHEAD shows the net working-tree result against HEAD, combining staged and
// unstaged changes once, plus untracked files. This is a read-only review.
func DiffHEAD(root, path string) (string, error) {
	return diffWorkingTree(root, path, false, true)
}

func diffWorkingTree(root, path string, staged, head bool) (string, error) {
	if path != "" {
		if err := literalPath(path); err != nil {
			return "", err
		}
	}
	args := []string{"diff", "--no-ext-diff", "--no-textconv"}
	unborn := false
	if staged {
		args = append(args, "--cached")
	}
	if head {
		if _, err := localCommand(root, "rev-parse", "--verify", "--quiet", "HEAD"); err == nil {
			args = append(args, "HEAD")
		} else {
			// Original gitDiff includes both index and subsequent working edits
			// when no HEAD exists yet.
			unborn = true
			args = append(args, "--cached")
		}
	}
	filter := path
	if filter == "" {
		filter = "."
	}
	args = append(args, "--", filter)
	out, err := localCommand(root, "--literal-pathspecs", args...)
	if unborn {
		unstaged, unstagedErr := localCommand(root, "--literal-pathspecs", "diff", "--no-ext-diff", "--no-textconv", "--", filter)
		if err != nil && unstagedErr != nil {
			return out, err
		}
		out += unstaged
		err = nil
	}
	if err != nil || staged {
		return out, err
	}
	changes, err := Changes(root)
	if err != nil {
		return "", err
	}
	for _, c := range changes {
		if c.Index != '?' || (path != "" && c.Path != path) {
			continue
		}
		patch, e := untrackedDiff(root, c.Path)
		if e != nil {
			if path != "" {
				return "", e
			}
			// Keep other readable changes visible and identify excluded content.
			patch = fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 100644\nPreview unavailable: %s\n", c.Path, c.Path, e)
		}
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += patch
	}
	return out, nil
}

func untrackedDiff(root, path string) (string, error) {
	if err := literalPath(path); err != nil {
		return "", err
	}
	p := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Lstat(p)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("untracked file is a link or not a regular file")
	}
	header := fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 100644\n", path, path)
	if info.Size() > 256*1024 {
		return header, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	if strings.ContainsRune(string(data), 0) {
		return header + fmt.Sprintf("Binary files /dev/null and b/%s differ\n", path), nil
	}
	if len(data) == 0 {
		return header, nil
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	patch := header + fmt.Sprintf("--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n+%s\n", path, len(lines), strings.Join(lines, "\n+"))
	if data[len(data)-1] != '\n' {
		patch += "\\ No newline at end of file\n"
	}
	return patch, nil
}

// StagedSummary lists the exact repository-wide paths included by a staged commit.
func StagedSummary(root string) (string, error) {
	return localCommand(root, "diff", "--cached", "--name-status", "--no-ext-diff", "--no-textconv")
}
