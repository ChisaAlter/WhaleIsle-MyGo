package workspacefs

import (
	"context"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// ListRemoteDir calls workspaceFiles/list when a dsh client is available.
// Empty path lists the workspace root.
func ListRemoteDir(ctx context.Context, client *dsh.Client, path string) ([]Entry, error) {
	if client == nil {
		return nil, ErrNotFile
	}
	path = strings.TrimSpace(path)
	var out api.WorkspaceDirectoryListing
	err := client.Call(ctx, "workspaceFiles/list", map[string]any{"path": path}, &out)
	if err != nil {
		return nil, err
	}
	ents := make([]Entry, 0, len(out.Entries))
	for _, e := range out.Entries {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			continue
		}
		isDir := strings.EqualFold(e.Type, "directory") || strings.EqualFold(e.Type, "dir")
		ents = append(ents, Entry{Name: name, IsDir: isDir})
	}
	return ents, nil
}

// StatRemote calls workspaceFiles/stat for a workspace-relative path.
func StatRemote(ctx context.Context, client *dsh.Client, path string) (api.WorkspaceFileStat, error) {
	var out api.WorkspaceFileStat
	if client == nil {
		return out, ErrNotFile
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return out, ErrNotFile
	}
	err := client.Call(ctx, "workspaceFiles/stat", map[string]any{"path": path}, &out)
	return out, err
}

// ReadRemoteBytes calls workspaceFiles/readBytes for a workspace-relative path.
func ReadRemoteBytes(ctx context.Context, client *dsh.Client, path string, length int) (api.WorkspaceFileBytes, error) {
	var out api.WorkspaceFileBytes
	if client == nil {
		return out, ErrNotFile
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return out, ErrNotFile
	}
	opts := api.WorkspaceByteReadOptions{}
	if length > 0 {
		lim := float64(length)
		opts.Range = &api.WorkspaceByteReadOptionsRange{Length: &lim}
	}
	err := client.Call(ctx, "workspaceFiles/readBytes", map[string]any{
		"path":    path,
		"options": opts,
	}, &out)
	return out, err
}

// ReadRemoteText calls workspaceFiles/read for a workspace-relative path.
func ReadRemoteText(ctx context.Context, client *dsh.Client, path string, limit int) (string, error) {
	if client == nil {
		return "", ErrNotFile
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ErrNotFile
	}
	args := map[string]any{"path": path}
	if limit > 0 {
		lim := float64(limit)
		args["range"] = api.WorkspaceFileRange{Limit: &lim}
	}
	var out api.WorkspaceFileText
	if err := client.Call(ctx, "workspaceFiles/read", args, &out); err != nil {
		return "", err
	}
	return out.Text, nil
}
