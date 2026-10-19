package session

import (
	"context"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
)

func resolveHistoryReferencePath(cwd, path string) string {
	absolute := strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\\`) || len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '/' || path[2] == '\\')
	if absolute || cwd == "" {
		return path
	}
	separator := "/"
	if (strings.HasPrefix(cwd, `\\`) || len(cwd) >= 2 && cwd[1] == ':') && strings.Contains(cwd, `\`) {
		separator = `\`
	}
	return strings.TrimRight(cwd, `/\`) + separator + strings.TrimLeft(path, `/\`)
}

// OpenHistoryReferenceOnHost is the waitable counterpart of the existing
// workspace path handoff. The caller invokes it only after explicit navigation
// was not handled by Files. It never reads/writes the global action banner.
func (s *Store) OpenHistoryReferenceOnHost(ctx context.Context, scope TurnFileScope, path string) error {
	s.mu.Lock()
	if scope.Binding == "" || scope != s.turnFilesScopeLocked() || s.client == nil {
		s.mu.Unlock()
		return fmt.Errorf("reference owner is no longer active")
	}
	client, endpoint := s.client, s.client.Endpoint
	captured := *client
	path = resolveHistoryReferencePath(s.cwd, path)
	s.mu.Unlock()
	current := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return ctx.Err() == nil && s.client == client && client.Endpoint == endpoint && scope == s.turnFilesScopeLocked()
	}
	if path == "" || strings.ContainsRune(path, 0) {
		return fmt.Errorf("invalid file reference")
	}
	if !current() {
		return fmt.Errorf("reference owner is no longer active")
	}
	var available api.CanOpenWorkspacePathResult
	if err := captured.Call(ctx, "session/canOpenWorkspacePath", map[string]any{}, &available); err != nil {
		return err
	}
	if !current() {
		return fmt.Errorf("reference owner is no longer active")
	}
	if !available {
		return fmt.Errorf("workspace path open unavailable")
	}
	var result api.SessionOpenWorkspacePathValue
	if err := captured.Call(ctx, "session/openWorkspacePath", map[string]any{"request": api.SessionOpenWorkspacePathRequest{Path: path}}, &result); err != nil {
		return err
	}
	if !current() {
		return fmt.Errorf("reference owner is no longer active")
	}
	if !result.Opened {
		return fmt.Errorf("workspace path was not opened")
	}
	return nil
}
