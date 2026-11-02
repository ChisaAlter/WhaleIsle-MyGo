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
	listed, err := listWorkspaceDirs(ctx, root, []string{rel})
	if err != nil {
		return nil, err
	}
	return listed[0], nil
}

// listWorkspaceDirs lists several directories with a single gitignore query.
// Whether a path is ignored depends only on the path, so asking about the
// children of many directories at once answers exactly what one query per
// directory would; the recursive index used to start one git process for each
// directory of the tree, which took half a minute on a few hundred of them.
func listWorkspaceDirs(ctx context.Context, root string, rels []string) ([][]Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	read := make([][]os.DirEntry, len(rels))
	var paths []string
	for i, rel := range rels {
		dir, err := JoinRoot(root, rel)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		read[i] = entries
		for _, entry := range entries {
			if !strings.EqualFold(entry.Name(), ".git") {
				paths = append(paths, filepath.ToSlash(filepath.Join(rel, entry.Name())))
			}
		}
	}
	ignored, err := gitIgnoredPaths(ctx, root, paths)
	if err != nil {
		return nil, err
	}
	out := make([][]Entry, len(rels))
	for i, rel := range rels {
		list := make([]Entry, 0, len(read[i]))
		for _, entry := range read[i] {
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
			list = append(list, Entry{Name: name, IsDir: entry.IsDir()})
		}
		sort.Slice(list, func(a, b int) bool {
			if list[a].IsDir != list[b].IsDir {
				return list[a].IsDir
			}
			return strings.ToLower(list[a].Name) < strings.ToLower(list[b].Name)
		})
		out[i] = list
	}
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
