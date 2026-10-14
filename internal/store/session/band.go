package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// BandAction is one pressable action from claudeCodeMods/watchBand.
type BandAction struct {
	ID    string
	Label string
}

// WatchBand opens claudeCodeMods/watchBand for the selected session.
func (s *Store) WatchBand(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	sessionID := s.selectedID
	if s.bandCancel != nil {
		s.bandCancel()
		s.bandCancel = nil
	}
	s.bandActions = nil
	s.bandGeneration = 0
	if mux == nil || sessionID == "" {
		s.mu.Unlock()
		s.fire()
		return
	}
	fctx, cancel := context.WithCancel(ctx)
	s.bandCancel = cancel
	s.mu.Unlock()
	go s.runBandWatch(fctx, mux, sessionID)
}

func (s *Store) runBandWatch(ctx context.Context, mux *dsh.Mux, sessionID string) {
	if err := mux.Dial(ctx); err != nil {
		return
	}
	stream, err := mux.Open(ctx, "claudeCodeMods/watchBand", map[string]any{
		"args": map[string]any{"agentId": sessionID},
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
		var snap api.SurfaceSnapshot
		if json.Unmarshal(raw, &snap) != nil {
			continue
		}
		actions := parseBandActions(snap.Tree)
		s.mu.Lock()
		if s.selectedID != sessionID {
			s.mu.Unlock()
			return
		}
		s.bandGeneration = snap.Generation
		s.bandActions = actions
		s.mu.Unlock()
		s.fire()
	}
}

// PressBand calls claudeCodeMods/pressBand for the current generation.
func (s *Store) PressBand(ctx context.Context, actionID string) {
	actionID = strings.TrimSpace(actionID)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	gen := s.bandGeneration
	s.mu.Unlock()
	if client == nil || sessionID == "" || actionID == "" {
		return
	}
	go func() {
		var out api.SurfaceSnapshot
		err := client.Call(ctx, "claudeCodeMods/pressBand", map[string]any{
			"agentId":    sessionID,
			"generation": gen,
			"actionId":   actionID,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = "band: " + err.Error()
		} else {
			s.bandGeneration = out.Generation
			s.bandActions = parseBandActions(out.Tree)
			s.lastAction = "band " + actionID
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func parseBandActions(tree []json.RawMessage) []BandAction {
	out := make([]BandAction, 0, len(tree))
	var walk func(raw json.RawMessage)
	walk = func(raw json.RawMessage) {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return
		}
		id := ""
		for _, key := range []string{"actionId", "id", "action"} {
			var s string
			if v, ok := obj[key]; ok && json.Unmarshal(v, &s) == nil && strings.TrimSpace(s) != "" {
				id = strings.TrimSpace(s)
				break
			}
		}
		label := id
		for _, key := range []string{"label", "title", "text", "name"} {
			var s string
			if v, ok := obj[key]; ok && json.Unmarshal(v, &s) == nil && strings.TrimSpace(s) != "" {
				label = strings.TrimSpace(s)
				break
			}
		}
		if id != "" {
			out = append(out, BandAction{ID: id, Label: label})
		}
		for _, key := range []string{"children", "nodes", "items", "tree"} {
			if chunk, ok := obj[key]; ok {
				var kids []json.RawMessage
				if json.Unmarshal(chunk, &kids) == nil {
					for _, kid := range kids {
						walk(kid)
					}
				}
			}
		}
	}
	for _, node := range tree {
		walk(node)
	}
	return out
}
