package session

import (
	"context"
	"encoding/json"
	"math"
	"strings"

	"whaleisle/internal/dsh/api"
)

// CanForkTurn accepts only a durable completed boundary whose last visible
// transcript contribution is an assistant answer. The Host remains authoritative.
func CanForkTurn(messages []Message, seq float64) bool {
	if seq <= 0 || math.IsNaN(seq) || math.IsInf(seq, 0) {
		return false
	}
	for i, msg := range messages {
		if msg.Seq != seq || msg.EventType != "turn/end" || msg.Streaming {
			continue
		}
		var end struct {
			Data struct {
				Reason struct {
					Kind string `json:"kind"`
				} `json:"reason"`
			} `json:"data"`
		}
		_ = json.Unmarshal(msg.Raw, &end)
		if end.Data.Reason.Kind == "error" {
			return false
		}
		for j := i - 1; j >= 0; j-- {
			prev := messages[j]
			if prev.Role == "user" || prev.EventType == "turn/end" {
				return false
			}
			if prev.Kind == MessageKindThinking || prev.Kind == MessageKindCompaction || prev.Kind == MessageKindDeliverables {
				continue
			}
			return prev.Role == "assistant" && !prev.Streaming && prev.Seq > 0 && prev.Seq < seq && strings.TrimSpace(prev.Body) != ""
		}
	}
	return false
}

// ForkTurn forks at the actual turn/end seq, never the assistant message seq.
// Selection changes invalidate the UI result, but do not undo server creation.
func (s *Store) ForkTurn(ctx context.Context, sessionID string, seq float64) {
	s.mu.Lock()
	if s.client == nil || ctx.Err() != nil || s.selectedID != sessionID || s.historyBranchBusy || !CanForkTurn(s.messages, seq) {
		s.mu.Unlock()
		return
	}
	client, generation := s.client, s.followGeneration
	s.historyBranchBusy = true
	s.lastAction = ""
	s.invalidateViewLocked()
	s.mu.Unlock()
	beforeSnapshotNotification("fork-start")
	s.fire()
	go func() {
		var out struct {
			SessionID string `json:"sessionId"`
		}
		err := client.Call(ctx, "session/fork", map[string]any{"request": api.SessionForkRequest{SessionId: sessionID, AtSeq: &seq}}, &out)
		s.mu.Lock()
		if s.client != client || s.selectedID != sessionID || s.followGeneration != generation {
			s.mu.Unlock()
			return
		}
		s.historyBranchBusy = false
		s.invalidateViewLocked()
		if err != nil {
			s.lastAction = "session/fork: " + err.Error()
			s.mu.Unlock()
			beforeSnapshotNotification("fork-complete")
			s.fire()
			return
		}
		if strings.TrimSpace(out.SessionID) == "" {
			s.lastAction = "session/fork: missing sessionId"
			s.mu.Unlock()
			beforeSnapshotNotification("fork-complete")
			s.fire()
			return
		}
		s.selectSessionLocked(out.SessionID)
		s.lastAction = "session forked"
		s.mu.Unlock()
		beforeSnapshotNotification("fork-complete")
		s.fire()
		s.RefreshList(ctx)
		s.FollowSelected(ctx)
	}()
}
