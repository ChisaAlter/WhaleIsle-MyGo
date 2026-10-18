package session

import (
	"context"
	"errors"
	"strings"

	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh/api"
)

// ErrNoClient is returned when directory picker RPCs run without a dsh client.
var ErrNoClient = errors.New("session: no dsh client")

// ListDirectories calls directoryPicker/list (host / remote cwd browser).
func (s *Store) ListDirectories(ctx context.Context, path string) (api.DirectoryListing, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	var out api.DirectoryListing
	if client == nil {
		return out, ErrNoClient
	}
	args := map[string]any{}
	if p := strings.TrimSpace(path); p != "" {
		args["path"] = p
	}
	err := client.Call(ctx, "directoryPicker/list", args, &out)
	return out, err
}

// PickDirectory calls directoryPicker/pick (host-native / remote picker).
func (s *Store) PickDirectory(ctx context.Context) (string, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return "", ErrNoClient
	}
	var out api.PickResult
	err := client.Call(ctx, "directoryPicker/pick", map[string]any{}, &out)
	return strings.TrimSpace(string(out)), err
}

// CreateDirectory calls directoryPicker/createDirectory under parent path.
func (s *Store) CreateDirectory(ctx context.Context, parent, name string) (string, error) {
	parent = strings.TrimSpace(parent)
	name = strings.TrimSpace(name)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return "", ErrNoClient
	}
	if !directorypicker.SelectablePath(parent) || name == "" {
		return "", errors.New("directory-create-failed: select a filesystem directory")
	}
	var out api.CreateDirectoryResult
	err := client.Call(ctx, "directoryPicker/createDirectory", map[string]any{
		"path": parent,
		"name": name,
	}, &out)
	return strings.TrimSpace(string(out)), err
}
