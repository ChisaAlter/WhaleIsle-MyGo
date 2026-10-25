package workspacefs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"whaleisle/internal/processutil"
)

// ListWorkspaceDir lists authorized immediate children. Dotfiles are visible;
// .git and gitignored entries follow the original desktop workspace contract.
// Symbolic-link directories are never traversed by the recursive index.
func ListWorkspaceDir(ctx context.Context, root, rel string) ([]Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := JoinRoot(root, rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !strings.EqualFold(entry.Name(), ".git") {
			paths = append(paths, filepath.ToSlash(filepath.Join(rel, entry.Name())))
		}
	}
	ignored, err := gitIgnoredPaths(ctx, root, paths)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := entry.Name()
		path := filepath.ToSlash(filepath.Join(rel, name))
		if strings.EqualFold(name, ".git") || ignored[path] {
			continue
		}
		// Keep normal internal links visible as files, but never expose escape or
		// metadata aliases. Entry.IsDir deliberately does not follow symlinks.
		if entry.Type()&os.ModeSymlink != 0 {
			if _, err := JoinRoot(root, path); err != nil {
				continue
			}
		}
		out = append(out, Entry{Name: name, IsDir: entry.IsDir()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func gitIgnoredPaths(ctx context.Context, root string, paths []string) (map[string]bool, error) {
	ignored := map[string]bool{}
	if len(paths) == 0 {
		return ignored, nil
	}
	command := exec.CommandContext(ctx, "git", "-C", root, "check-ignore", "--no-index", "--stdin", "-z")
	processutil.Background(command)
	command.Stdin = strings.NewReader(strings.Join(paths, "\x00") + "\x00")
	output, err := command.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		var exit *exec.ExitError
		// Git missing, non-repository and unsupported Git all fall back to the
		// unfiltered listing, as the original workspace implementation does.
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return ignored, nil
		}
	}
	for _, path := range strings.Split(string(output), "\x00") {
		if path != "" {
			ignored[path] = true
		}
	}
	return ignored, nil
}
