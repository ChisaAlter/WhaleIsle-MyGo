package im

import (
	"context"
	"errors"
	"strings"
	"whaleisle/internal/dsh/api"
)

// Directory navigation uses the current Host's actual directory picker. It
// never scans the desktop filesystem or changes the application's workspace.
func (s *Store) ListDirectories(ctx context.Context, path string) (api.DirectoryListing, error) {
	s.mu.Lock()
	client, epoch := s.client, s.connectionEpoch
	s.mu.Unlock()
	var out api.DirectoryListing
	if client == nil {
		return out, errors.New("Host is not connected")
	}
	args := map[string]any{}
	if strings.TrimSpace(path) != "" {
		args["path"] = strings.TrimSpace(path)
	}
	err := client.Call(ctx, "directoryPicker/list", args, &out)
	s.mu.Lock()
	current := s.client == client && s.connectionEpoch == epoch
	s.mu.Unlock()
	if !current {
		return api.DirectoryListing{}, errors.New("Host changed; reopen the directory picker")
	}
	return out, err
}

func (s *Store) PickDirectory(ctx context.Context) (string, error) {
	s.mu.Lock()
	client, epoch := s.client, s.connectionEpoch
	s.mu.Unlock()
	if client == nil {
		return "", errors.New("Host is not connected")
	}
	var out api.PickResult
	err := client.Call(ctx, "directoryPicker/pick", map[string]any{}, &out)
	s.mu.Lock()
	current := s.client == client && s.connectionEpoch == epoch
	s.mu.Unlock()
	if !current {
		return "", errors.New("Host changed; reopen the directory picker")
	}
	return strings.TrimSpace(out), err
}
