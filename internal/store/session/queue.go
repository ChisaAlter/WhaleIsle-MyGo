package session

import (
	"context"
	"encoding/json"
)

// UpdateQueueAction is a session/updateQueue action discriminant.
type UpdateQueueAction string

const (
	QueueActionRemove UpdateQueueAction = "remove"
	QueueActionSteer  UpdateQueueAction = "steer"
	QueueActionEdit   UpdateQueueAction = "edit"
)

// UpdateQueue calls session/updateQueue for the selected session (or sessionID).
// For QueueActionEdit, pass the new text as editText (ignored for other actions).
func (s *Store) UpdateQueue(ctx context.Context, sessionID, itemID string, action UpdateQueueAction, editText ...string) {
	if s == nil || itemID == "" || action == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	if sessionID == "" {
		sessionID = s.selectedID
	}
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		s.mu.Lock()
		s.lastAction = "queue: no session"
		s.mu.Unlock()
		s.fire()
		return
	}
	payload := map[string]any{"type": string(action)}
	if action == QueueActionEdit && len(editText) > 0 {
		payload["text"] = editText[0]
	}
	act, _ := json.Marshal(payload)
	req := map[string]any{
		"sessionId": sessionID,
		"itemId":    itemID,
		"action":    json.RawMessage(act),
	}
	go func() {
		var out any
		err := client.Call(ctx, "session/updateQueue", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = "queue: " + err.Error()
		} else {
			s.lastAction = "queue " + string(action) + " " + itemID
			// Optimistic local remove so the plan strip updates before follow.
			if action == QueueActionRemove {
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
	}()
}
