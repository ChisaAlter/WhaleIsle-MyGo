package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
)

// WatchControl opens session/control and records the latest control frame type.
func (s *Store) WatchControl(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	sessionID := s.selectedID
	if s.controlCancel != nil {
		s.controlCancel()
		s.controlCancel = nil
	}
	s.controlNote = ""
	if mux == nil || sessionID == "" {
		s.mu.Unlock()
		s.fire()
		return
	}
	fctx, cancel := context.WithCancel(ctx)
	s.controlCancel = cancel
	s.mu.Unlock()
	go s.runControlWatch(fctx, mux, sessionID)
}

func (s *Store) runControlWatch(ctx context.Context, mux *dsh.Mux, sessionID string) {
	if err := mux.Dial(ctx); err != nil {
		return
	}
	stream, err := mux.Open(ctx, "session/control", map[string]any{
		"args": map[string]any{
			"request": map[string]any{"sessionId": sessionID},
		},
	})
	if err != nil {
		return
	}
	defer stream.Cancel()

	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, dsh.ErrStreamEnd) || errors.Is(err, context.Canceled) ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			return
		}
		var head struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(raw, &head)
		note := strings.TrimSpace(head.Type)
		if note == "" {
			note = truncateRunes(string(raw), 80)
		}
		s.mu.Lock()
		if s.selectedID != sessionID {
			s.mu.Unlock()
			return
		}
		s.controlNote = note
		s.mu.Unlock()
		s.fire()
	}
}
