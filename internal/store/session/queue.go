package session

import (
	"context"
	"strings"
	"whaleisle/internal/dsh"
)

// UpdateQueueAction is the HostAPI queue action's kind discriminant.
type UpdateQueueAction string

const (
	QueueActionRemove UpdateQueueAction = "remove"
	QueueActionSteer  UpdateQueueAction = "steer"
	QueueActionEdit   UpdateQueueAction = "edit"
)

// queueActionPayload follows session-controller QueueAction: kind, and text
// content blocks for edits. Never submit preview text as an edit implicitly.
func queueActionPayload(action UpdateQueueAction, editText []string) (map[string]any, bool) {
	payload := map[string]any{"kind": string(action)}
	switch action {
	case QueueActionRemove, QueueActionSteer:
	case QueueActionEdit:
		if len(editText) == 0 || strings.TrimSpace(editText[0]) == "" {
			return nil, false
		}
		payload["content"] = []map[string]string{{"type": "text", "text": editText[0]}}
	default:
		return nil, false
	}
	return payload, true
}

// UpdateQueue mutates one pending item; only an accepted Host receipt changes
// the local queue. Responses belonging to a previous follow are ignored.
func (s *Store) UpdateQueue(ctx context.Context, sessionID, itemID string, action UpdateQueueAction, editText ...string) {
	if s == nil || itemID == "" {
		return
	}
	payload, valid := queueActionPayload(action, editText)
	if !valid {
		return
	}
	s.mu.Lock()
	client := s.client
	if sessionID == "" {
		sessionID = s.selectedID
	}
	generation := s.followGeneration
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		s.mu.Lock()
		s.lastAction = "queue: no session"
		s.mu.Unlock()
		s.fire()
		return
	}
	req := map[string]any{"sessionId": sessionID, "itemId": itemID, "action": payload}
	go s.runUpdateQueue(ctx, client, sessionID, generation, itemID, action, req)
}

func (s *Store) runUpdateQueue(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, itemID string, action UpdateQueueAction, req map[string]any) {
	var out struct {
		Accepted bool `json:"accepted"`
	}
	err := client.Call(ctx, "session/updateQueue", map[string]any{"request": req}, &out)
	s.mu.Lock()
	if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.lastAction = "queue: " + err.Error()
	} else if !out.Accepted {
		s.lastAction = "queue: mutation was not accepted"
	} else {
		s.lastAction = "queue " + string(action) + " " + itemID
		// Steer consumes the pending next-turn occurrence too.
		if action == QueueActionRemove || action == QueueActionSteer {
			next := make([]PlanQueueItem, 0, len(s.planBoard.Queue))
			for _, it := range s.planBoard.Queue {
				if it.ID != itemID {
					next = append(next, it)
				}
			}
			s.planBoard.Queue = next
		}
	}
	s.mu.Unlock()
	s.fire()
}
