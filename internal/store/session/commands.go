package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
)

// ListSlashCommands calls commands/list for the selected session when a client is attached.
func (s *Store) ListSlashCommands(ctx context.Context) []api.ListResultItem2 {
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return nil
	}
	var out api.ListResult3
	err := client.Call(ctx, "commands/list", map[string]any{
		"agentId": sessionID,
	}, &out)
	if err != nil {
		return nil
	}
	return out
}

// ExecuteCommandLine runs commands/execute for the selected session.
func (s *Store) ExecuteCommandLine(ctx context.Context, line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return nil
	}
	var result api.ExecuteResult
	return client.Call(ctx, "commands/execute", map[string]any{
		"agentId": sessionID,
		"line":    line,
	}, &result)
}
