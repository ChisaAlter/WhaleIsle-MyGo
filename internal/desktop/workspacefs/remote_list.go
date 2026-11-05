package workspacefs

import (
	"context"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// ListRemoteDir calls workspaceFiles/list when a dsh client is available.
// Empty path lists the workspace root.
func ListRemoteDir(ctx context.Context, client *dsh.Client, path string) ([]Entry, error) {
	return listRemoteDir(ctx, client, path, "")
}

// ListRemoteDirScoped binds listings to the same Session as directory watches.
func ListRemoteDirScoped(ctx context.Context, client *dsh.Client, sessionID, path string) ([]Entry, error) {
	return listRemoteDir(ctx, client, path, sessionID)
}

func listRemoteDir(ctx context.Context, client *dsh.Client, path, sessionID string) ([]Entry, error) {
	if client == nil {
		return nil, ErrNotFile
	}
	// DirectoryTree names its root with an empty relative path. The scoped
	// Host service requires a nonempty path and resolves "." against that Session.
	if path == "" && sessionID != "" {
		path = "."
	}
	var out api.WorkspaceDirectoryListing
	args := map[string]any{"path": path}
	if sessionID != "" {
		args["workspaceFileScopeId"] = sessionID
	}
	err := client.Call(ctx, "workspaceFiles/list", args, &out)
	if err != nil {
		return nil, err
	}
	ents := make([]Entry, 0, len(out.Entries))
	for _, e := range out.Entries {
		name := e.Name
		if name == "" {
			continue
		}
		isDir := strings.EqualFold(e.Type, "directory") || strings.EqualFold(e.Type, "dir")
		ents = append(ents, Entry{Name: name, IsDir: isDir, Other: strings.EqualFold(e.Type, "other")})
	}
	return ents, nil
}

// StatRemote calls workspaceFiles/stat for a workspace-relative path.
func StatRemote(ctx context.Context, client *dsh.Client, path string) (api.WorkspaceFileStat, error) {
	return statRemote(ctx, client, path, "")
}

// StatRemoteScoped binds supplemental file metadata to the captured Session.
func StatRemoteScoped(ctx context.Context, client *dsh.Client, sessionID, path string) (api.WorkspaceFileStat, error) {
	if strings.TrimSpace(sessionID) == "" {
		return api.WorkspaceFileStat{}, errors.New("Host file stat requires a session scope")
	}
	return statRemote(ctx, client, path, sessionID)
}

func statRemote(ctx context.Context, client *dsh.Client, path, sessionID string) (api.WorkspaceFileStat, error) {
	var out api.WorkspaceFileStat
	if client == nil {
		return out, ErrNotFile
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return out, ErrNotFile
	}
	args := map[string]any{"path": path}
	if sessionID != "" {
		args["workspaceFileScopeId"] = sessionID
	}
	err := client.Call(ctx, "workspaceFiles/stat", args, &out)
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
