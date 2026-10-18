package git

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DiscardPlan is a review of exactly one file. Its private token binds the
// repository, HEAD, selected index entry, status, and on-disk contents.
type DiscardPlan struct {
	Path        string
	Untracked   bool
	root, token string
}

func discardPath(root, path string) (string, string, error) {
	if cleanWorkspaceRoot(root) == "" {
		return "", "", ErrNotRepo
	}
	if err := literalPath(path); err != nil {
		return "", "", err
	}
	rel := filepath.FromSlash(path)
	if filepath.Clean(rel) != rel || strings.Contains(path, "\\") || strings.Contains(path, ":") {
		return "", "", errors.New("select a literal workspace-relative file")
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.EqualFold(part, ".git") {
			return "", "", errors.New("Git metadata cannot be discarded")
		}
	}
	canonical, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	canonical, err = filepath.EvalSymlinks(canonical)
	if err != nil {
		return "", "", err
	}
	current := canonical
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) && i == len(parts)-1 {
			break
		}
		if err != nil {
			return "", "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", errors.New("linked paths cannot be discarded")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return "", "", errors.New("invalid parent directory")
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return "", "", errors.New("select one regular file; directories and submodules cannot be discarded")
		}
	}
	return canonical, filepath.Join(canonical, rel), nil
}

// PrepareDiscard reads state only. Canceling by dropping this plan changes no files.
func PrepareDiscard(root, path string) (DiscardPlan, error) {
	canonical, full, err := discardPath(root, path)
	if err != nil {
		return DiscardPlan{}, err
	}
	changes, err := Changes(canonical)
	if err != nil {
		return DiscardPlan{}, err
	}
	var selected *Change
	for _, change := range changes {
		if change.Path == filepath.ToSlash(path) {
			c := change
			selected = &c
			break
		}
	}
	if selected == nil || !selected.Unstaged() {
		return DiscardPlan{}, errors.New("file has no unstaged changes; refresh before reviewing")
	}
	xy := string([]byte{selected.Index, selected.Worktree})
	if strings.ContainsAny(xy, "U") || xy == "AA" || xy == "DD" {
		return DiscardPlan{}, errors.New("resolve this merge conflict explicitly before discarding")
	}
	gitDir, err := localCommand(canonical, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return DiscardPlan{}, err
	}
	head, err := localCommand(canonical, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		head = "detached"
	}
	oid, _ := localCommand(canonical, "rev-parse", "--verify", "HEAD")
	index, err := localCommand(canonical, "--literal-pathspecs", "ls-files", "--stage", "--", path)
	if err != nil {
		return DiscardPlan{}, err
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00", canonical, gitDir, head, oid, path, xy, selected.Original, index)
	info, err := os.Lstat(full)
	if os.IsNotExist(err) {
		if xy == "??" {
			return DiscardPlan{}, errors.New("file is missing")
		}
		io.WriteString(h, "missing")
	} else if err != nil {
		return DiscardPlan{}, err
	} else {
		if !info.Mode().IsRegular() {
			return DiscardPlan{}, errors.New("file changed; review again")
		}
		fmt.Fprintf(h, "%d\x00", info.Mode())
		file, err := os.Open(full)
		if err != nil {
			return DiscardPlan{}, err
		}
		opened, statErr := file.Stat()
		if statErr != nil || !os.SameFile(info, opened) {
			file.Close()
			return DiscardPlan{}, errors.New("file changed; review again")
		}
		_, readErr := io.Copy(h, file)
		file.Close()
		if readErr != nil {
			return DiscardPlan{}, readErr
		}
	}
	return DiscardPlan{Path: path, Untracked: xy == "??", root: canonical, token: hex.EncodeToString(h.Sum(nil))}, nil
}

// DiscardReviewed restores one tracked file from the index (preserving staged
// changes), or removes exactly one reviewed untracked file. It never resets the
// repository, recurses into a directory, or accepts an unchecked path list.
func DiscardReviewed(root string, plan DiscardPlan) (string, error) {
	if plan.token == "" {
		return "", errors.New("review the selected file before discarding")
	}
	current, err := PrepareDiscard(root, plan.Path)
	if err != nil {
		return "", err
	}
	if current.root != plan.root || current.token != plan.token || current.Untracked != plan.Untracked {
		return "", errors.New("repository or file changed since review; review again before discarding")
	}
	if plan.Untracked {
		// The literal, checked leaf is used directly: there is no recursive removal
		// or Git pathspec expansion, including for names containing wildcard syntax.
		_, full, err := discardPath(root, plan.Path)
		if err != nil {
			return "", err
		}
		if err = os.Remove(full); err != nil {
			return "", err
		}
		return plan.Path, nil
	}
	return localCommand(root, "--literal-pathspecs", "checkout", "--", plan.Path)
}
