package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
)

func (s *Store) setPinnedSessionIDs(ids []string) {
	s.mu.Lock()
	s.pinnedSessionIDs = append([]string(nil), ids...)
	s.applyPinnedFlagsLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) applyPinnedFlagsLocked() {
	pinned := map[string]struct{}{}
	for _, id := range s.pinnedSessionIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			pinned[id] = struct{}{}
		}
	}
	for i := range s.items {
		_, s.items[i].Pinned = pinned[s.items[i].SessionID]
	}
}

// PinSession calls workspace/pinSession for a session id.
func (s *Store) PinSession(ctx context.Context, sessionID string) {
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
		req := map[string]any{"sessionId": sessionID}
		var out api.WorkspacePinValue
		err := client.Call(ctx, "workspace/pinSession", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.pinnedSessionIDs = append([]string(nil), out.PinnedSessionIds...)
		s.applyPinnedFlagsLocked()
		s.lastAction = "session pinned"
		s.mu.Unlock()
		s.fire()
	}()
}

// UnpinSession calls workspace/unpinSession for a session id.
func (s *Store) UnpinSession(ctx context.Context, sessionID string) {
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
		req := map[string]any{"sessionId": sessionID}
		var out api.WorkspacePinValue
		err := client.Call(ctx, "workspace/unpinSession", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.pinnedSessionIDs = append([]string(nil), out.PinnedSessionIds...)
		s.applyPinnedFlagsLocked()
		s.lastAction = "session unpinned"
		s.mu.Unlock()
		s.fire()
	}()
}
