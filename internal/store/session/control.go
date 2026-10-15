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
	client := s.client
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
	go s.runControlWatch(fctx, client, mux, sessionID)
}

func (s *Store) controlWatchCurrentLocked(ctx context.Context, client *dsh.Client, mux *dsh.Mux, sessionID string) bool {
	return ctx.Err() == nil && s.selectedID == sessionID && s.client == client && s.mux == mux
}

func (s *Store) runControlWatch(ctx context.Context, client *dsh.Client, mux *dsh.Mux, sessionID string) {
	if err := mux.Dial(ctx); err != nil {
		return
	}
	stream, err := mux.Open(ctx, "session/control", map[string]any{"args": map[string]any{}})
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
		if !s.controlWatchCurrentLocked(ctx, client, mux, sessionID) {
			s.mu.Unlock()
			return
		}
		s.applyAgentNavigationControlLocked(raw)
		s.applyCostControlLocked(raw)
		s.applyQueueControlLocked(raw)
		s.applyQuestionControlLocked(raw)
		s.applyPermissionControlLocked(raw)
		s.applyModelControlLocked(raw)
		s.applyHistoryOutlineControlLocked(raw)
		s.controlNote = note
		s.mu.Unlock()
		s.fire()
	}
}
