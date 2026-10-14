package session

import (
	"context"
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh/api"
)

// RefreshProjections loads session/projections for the selected session and
// applies the title when present (list projections may lag behind follow).
func (s *Store) RefreshProjections(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go func() {
		var out api.SessionProjectionsValue
		err := client.Call(ctx, "session/projections", map[string]any{
			"request": map[string]any{"sessionId": sessionID},
		}, &out)
		if err != nil {
			return
		}
		title := titleFromProjections(json.RawMessage(out.Values))

		s.mu.Lock()
		if s.selectedID != sessionID || s.client != client {
			s.mu.Unlock()
			return
		}
		s.applyCostBaselineLocked(sessionID, out.AsOfSeq, out.Values)
		for i := range s.items {
			if s.items[i].SessionID == sessionID {
				if title != "" && (strings.TrimSpace(s.items[i].Title) == "" || s.items[i].Title == sessionID) {
					s.items[i].Title = title
				}
				break
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}
