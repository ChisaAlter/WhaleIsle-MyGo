package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh"
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
	client, generation := s.client, s.eventsGeneration
	s.mu.Unlock()
	if client == nil {
		return
	}
	go s.runUnarchiveSession(ctx, client, generation, sessionID)
}

// Share the real restore RPC while retaining the Host captured at admission.
func (s *Store) runUnarchiveSession(ctx context.Context, client *dsh.Client, generation uint64, sessionID string) {
	var out api.WorkspaceArchiveValue
	err := client.Call(ctx, "workspace/unarchiveSession", map[string]any{
		"request": map[string]any{"sessionId": sessionID},
	}, &out)
	s.mu.Lock()
	if s.client != client || s.eventsGeneration != generation {
		s.mu.Unlock()
		return
	}
	if err != nil {
		if s.lastAction != err.Error() {
			s.invalidateViewLocked()
		}
		s.lastAction = err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	s.archivedSessionIDs = append([]string(nil), out.ArchivedSessionIds...)
	s.lastAction = "session unarchived"
	s.invalidateViewLocked()
	if s.archiveUndo != nil && s.archiveUndo.notice.SessionID == sessionID {
		s.archiveUndo = nil
	}
	// Admit the refresh under the same Host lock as the acknowledgement.
	// A repaint may synchronously attach another Host.
	var listGeneration uint64
	if ctx.Err() == nil {
		_, listGeneration = s.beginListRefreshLocked()
	}
	s.mu.Unlock()
	s.fire()
	if listGeneration != 0 {
		s.runRefreshList(ctx, client, listGeneration)
	}
}

// DeleteSession calls session/delete for a session id and refreshes the list.
func (s *Store) DeleteSession(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	client, generation := s.client, s.eventsGeneration
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
		if s.client != client || s.eventsGeneration != generation {
			s.mu.Unlock()
			return
		}
		if err != nil {
			if s.lastAction != err.Error() {
				s.invalidateViewLocked()
			}
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.applyDeleteResultLocked(out)
		s.lastAction = "session deleted"
		// Admit the refresh under the same Host lock as the acknowledgement.
		// A repaint may synchronously attach another Host.
		var listGeneration uint64
		if ctx.Err() == nil {
			_, listGeneration = s.beginListRefreshLocked()
		}
		s.mu.Unlock()
		s.fire()
		if listGeneration != 0 {
			go s.runRefreshList(ctx, client, listGeneration)
		}
	}()
}
