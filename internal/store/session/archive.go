package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
)

// ArchiveSession first asks the Host to archive without stopping any activity.
func (s *Store) ArchiveSession(ctx context.Context, sessionID string) {
	s.beginArchive(ctx, sessionID)
}

// UnarchiveSession calls workspace/unarchiveSession and refreshes the list.
func (s *Store) UnarchiveSession(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var out api.WorkspaceArchiveValue
		err := client.Call(ctx, "workspace/unarchiveSession", map[string]any{
			"request": map[string]any{"sessionId": sessionID},
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.archivedSessionIDs = append([]string(nil), out.ArchivedSessionIds...)
		s.lastAction = "session unarchived"
		s.mu.Unlock()
		s.fire()
		s.RefreshList(ctx)
	}()
}

// DeleteSession calls session/delete for a session id and refreshes the list.
func (s *Store) DeleteSession(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var out api.SessionDeleteValue
		err := client.Call(ctx, "session/delete", map[string]any{
			"request": map[string]any{"sessionId": sessionID},
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.applyDeleteResultLocked(out)
		s.lastAction = "session deleted"
		s.mu.Unlock()
		s.fire()
		s.RefreshList(ctx)
	}()
}
