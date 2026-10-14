package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
)

// ArchiveSession calls workspace/archiveSession and refreshes the session list.
func (s *Store) ArchiveSession(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	wasSelected := s.selectedID == sessionID
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		stop := true
		req := api.WorkspaceArchiveSessionRequest{
			SessionId:    sessionID,
			StopActivity: &stop,
		}
		var out api.WorkspaceArchiveValue
		err := client.Call(ctx, "workspace/archiveSession", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		if wasSelected {
			s.selectedID = ""
		}
		s.archivedSessionIDs = append([]string(nil), out.ArchivedSessionIds...)
		s.lastAction = "session archived"
		s.mu.Unlock()
		s.fire()
		s.RefreshList(ctx)
	}()
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
