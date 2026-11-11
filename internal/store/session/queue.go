package session

import (
	"context"
	"errors"
	"strings"
	"whaleisle/internal/dsh"
)

// UpdateQueueAction is the HostAPI queue action's kind discriminant.
type UpdateQueueAction string

const (
	QueueActionRemove UpdateQueueAction = "remove"
	QueueActionSteer  UpdateQueueAction = "steer"
	QueueActionEdit   UpdateQueueAction = "edit"
	QueueActionMove   UpdateQueueAction = "move"
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

// MoveQueue reorders one pending item to toIndex via session/updateQueue action=move.
func (s *Store) MoveQueue(ctx context.Context, sessionID, itemID string, toIndex int) {
	if s == nil || itemID == "" || toIndex < 0 {
		return
	}
	s.mu.Lock()
	client := s.client
	if sessionID == "" {
		sessionID = s.selectedID
	}
	generation := s.followGeneration
	if client != nil && sessionID != "" {
		state := s.queueActionsLocked()
		if sessionID != s.selectedID || !state.Can(QueueActionMove, s.queueItemLocked(itemID)) || toIndex >= len(s.planBoard.Queue) {
			s.mu.Unlock()
			return
		}
		fromIndex := -1
		for i, it := range s.planBoard.Queue {
			if it.ID == itemID {
				fromIndex = i
				break
			}
		}
		if fromIndex == toIndex {
			s.mu.Unlock()
			return
		}
		s.queueMutationScope, s.queueMutationID = state.Scope, itemID
		s.invalidateViewLocked()
	}
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		s.mu.Lock()
		s.lastAction = "queue: no session"
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	payload := map[string]any{"kind": string(QueueActionMove), "toIndex": toIndex}
	req := map[string]any{"sessionId": sessionID, "itemId": itemID, "action": payload}
	s.fire()
	go s.runUpdateQueue(ctx, client, sessionID, generation, itemID, QueueActionMove, req)
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
	if client != nil && sessionID != "" {
		state := s.queueActionsLocked()
		if sessionID != s.selectedID || !state.Can(action, s.queueItemLocked(itemID)) {
			s.mu.Unlock()
			return
		}
		s.queueMutationScope, s.queueMutationID = state.Scope, itemID
		s.invalidateViewLocked()
	}
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		s.mu.Lock()
		s.lastAction = "queue: no session"
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	req := map[string]any{"sessionId": sessionID, "itemId": itemID, "action": payload}
	s.fire()
	go s.runUpdateQueue(ctx, client, sessionID, generation, itemID, action, req)
}

func (s *Store) runUpdateQueue(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, itemID string, action UpdateQueueAction, req map[string]any, composerBatch ...bool) error {
	batch := len(composerBatch) > 0 && composerBatch[0]
	var out struct {
		Accepted bool `json:"accepted"`
	}
	err := client.Call(ctx, "session/updateQueue", map[string]any{"request": req}, &out)
	if err == nil && !out.Accepted {
		err = errors.New("mutation was not accepted")
	}
	s.mu.Lock()
	if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) {
		s.mu.Unlock()
		return context.Canceled
	}
	if !batch && s.queueMutationScope == s.queueActionsLocked().Scope && s.queueMutationID == itemID {
		s.queueMutationScope, s.queueMutationID = "", ""
	}
	if err != nil {
		// The original conversation.updateQueue treats a consumed occurrence or
		// an unavailable steering window as convergence for every Steer gesture,
		// including the per-row button. Other mutations still report failures.
		if action != QueueActionSteer || !queueSteerConverged(err) {
			s.lastAction = "queue: " + err.Error()
		}
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
		} else if action == QueueActionMove {
			if actMap, ok := req["action"].(map[string]any); ok {
				if toIndex, ok := actMap["toIndex"].(int); ok && toIndex >= 0 && toIndex < len(s.planBoard.Queue) {
					fromIndex := -1
					for i, it := range s.planBoard.Queue {
						if it.ID == itemID {
							fromIndex = i
							break
						}
					}
					if fromIndex >= 0 && fromIndex != toIndex {
						moved := s.planBoard.Queue[fromIndex]
						rest := make([]PlanQueueItem, 0, len(s.planBoard.Queue)-1)
						for i, it := range s.planBoard.Queue {
							if i != fromIndex {
								rest = append(rest, it)
							}
						}
						reordered := make([]PlanQueueItem, 0, len(s.planBoard.Queue))
						reordered = append(reordered, rest[:toIndex]...)
						reordered = append(reordered, moved)
						reordered = append(reordered, rest[toIndex:]...)
						s.planBoard.Queue = reordered
					}
				}
			}
		}
	}
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	return err
}
