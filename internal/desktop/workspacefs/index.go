package workspacefs

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// RemoteDirectoryLister preserves the existing Host workspaceFiles/list route
// while allowing a cancelled/obsolete search to stop its outstanding request.
type RemoteDirectoryLister func(context.Context, string) ([]Entry, error)

// IndexRelPaths returns every visible slash-relative path, including directories.
// maxFiles is retained for source compatibility, but no longer truncates the
// index: callers must page presentation rather than permanently omit targets.
func IndexRelPaths(root string, maxFiles int) ([]string, error) {
	return walkRelPaths(context.Background(), root, "", nil, true)
}

// SearchRelPaths traverses the complete visible tree off the UI thread. It
// returns files only, matching the original recursive project-file search.
func SearchRelPaths(ctx context.Context, root, query string, remote RemoteDirectoryLister) ([]string, error) {
	return walkRelPaths(ctx, root, normalizeFileSearch(query), remote, false)
}

func walkRelPaths(ctx context.Context, root, needle string, remote RemoteDirectoryLister, includeDirs bool) ([]string, error) {
	return walkRelPathsWithSource(ctx, root, needle, remote, includeDirs, false)
}

func walkRelPathsWithSource(ctx context.Context, root, needle string, remote RemoteDirectoryLister, includeDirs, hostOnly bool) ([]string, error) {
	var out []string
	pending := []string{""}
	// Choose the local or Host source once at the root. A failed nested local
	// read must not accidentally jump into a different Host workspace.
	local := !hostOnly
	var first []Entry
	var err error
	if hostOnly {
		if remote == nil {
			return nil, fmt.Errorf("workspace Host directory reader is unavailable")
		}
		first, err = remote(ctx, "")
	} else {
		first, err = ListWorkspaceDir(ctx, root, "")
	}
	if local && err != nil && ctx.Err() == nil && remote != nil {
		first, err = remote(ctx, "")
		local = false
	}
	if err != nil {
		return nil, err
	}
	if local {
		// A local tree is walked a level at a time, asking git about the whole level at
		// once. The remote lister below stays one request per directory.
		return walkLocalLevels(ctx, root, needle, first, includeDirs)
	}
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rel := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		entries := first
		if rel != "" {
			if local {
				entries, err = ListWorkspaceDir(ctx, root, rel)
			} else {
				entries, err = remote(ctx, rel)
			}
		}
		if err != nil {
			return nil, err
		}
		entries, err = validatedEntries(entries)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.Other {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			name := entry.Name
			path := name
			if rel != "" {
				path = rel + "/" + name
			}
			if entry.IsDir {
				pending = append(pending, path)
			}
			if (includeDirs || !entry.IsDir) && matchesFileSearch(path, needle) {
				out = append(out, path)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out, nil
}

func validatedEntries(entries []Entry) ([]Entry, error) {
	out := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") || filepath.IsAbs(name) {
			return nil, fmt.Errorf("workspacefs: invalid directory entry %q", name)
		}
		if !strings.EqualFold(name, ".git") {
			out = append(out, entry)
		}
	}
	return out, nil
}

// walkLocalLevels is walkRelPaths for a local tree. first is the listing of the
// root. Each level of directories is listed with one gitignore query, so a tree
// of hundreds of directories costs a handful of git processes instead of one
// per directory. The set of paths is the same: ignored directories are still
// never entered, and the result is sorted.
func walkLocalLevels(ctx context.Context, root, needle string, first []Entry, includeDirs bool) ([]string, error) {
	var out []string
	level := []string{""}
	listed := [][]Entry{first}
	for len(level) > 0 {
		var next []string
		for i, rel := range level {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			entries, err := validatedEntries(listed[i])
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if entry.Other {
					continue
				}
				path := entry.Name
				if rel != "" {
					path = rel + "/" + entry.Name
				}
				if entry.IsDir {
					next = append(next, path)
				}
				if (includeDirs || !entry.IsDir) && matchesFileSearch(path, needle) {
					out = append(out, path)
				}
			}
		}
		level = next
		if len(level) == 0 {
			break
		}
		var err error
		if listed, err = listWorkspaceDirs(ctx, root, level); err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out, nil
}
