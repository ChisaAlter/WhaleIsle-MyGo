package session

import (
	"context"
	"errors"
	"whaleisle/internal/dsh"
)

type composerQueueSteer struct {
	client                        *dsh.Client
	sessionID, scope, reservation string
	epoch, follow                 uint64
	ids                           []string
}

func queueSteerConverged(err error) bool {
	var rpc *dsh.Error
	return errors.As(err, &rpc) && (rpc.Code == "session/steer-unavailable" || rpc.Code == "session/queue-item-not-found")
}

// SteerComposerQueue handles only an explicit empty-editor accelerated gesture.
// It snapshots the current Host inbox IDs and reuses the per-row Steer request
// and receipt path in FIFO order. It never resends text or creates a prompt.
func (s *Store) SteerComposerQueue(ctx context.Context, sessionID string, hostGeneration uint64) {
	if s == nil || ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	op := s.beginComposerQueueSteerLocked(sessionID, hostGeneration)
	s.mu.Unlock()
	if op == nil {
		return
	}
	s.fire()
	go s.runComposerQueueSteer(ctx, op)
}

func (s *Store) beginComposerQueueSteerLocked(sessionID string, hostGeneration uint64) *composerQueueSteer {
	if s.client == nil || sessionID == "" || s.selectedID != sessionID || s.composerHostGeneration != hostGeneration || s.queueProjectionSessionID != sessionID || s.queueProjectionGeneration != s.followGeneration {
		return nil
	}
	state := s.queueActionsLocked()
	// Queue rows may survive a Host switch while its new baseline loads.
	// Only the current follow or current Host roster is running evidence.
	running := s.turnStartMs > 0
	if s.composerRosterClient == s.client {
		for _, item := range s.items {
			if item.SessionID == sessionID && item.Running {
				running = true
			}
		}
	}
	if !state.Mutable || state.Busy || !running {
		return nil
	}
	ids := make([]string, 0, len(s.planBoard.Queue))
	seen := map[string]bool{}
	for _, item := range s.planBoard.Queue {
		if item.ID != "" && !seen[item.ID] {
			ids = append(ids, item.ID)
			seen[item.ID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	op := &composerQueueSteer{client: s.client, sessionID: sessionID, scope: state.Scope, reservation: "composer:" + newRequestID(), epoch: hostGeneration, follow: s.followGeneration, ids: ids}
	s.queueMutationScope, s.queueMutationID = op.scope, op.reservation
	return op
}

func (s *Store) composerQueueSteerCurrentLocked(ctx context.Context, op *composerQueueSteer) bool {
	s.invalidateViewLocked()
	return s.composerHostGeneration == op.epoch && s.sessionResultCurrentLocked(ctx, op.client, op.sessionID, op.follow) && s.queueMutationScope == op.scope && s.queueMutationID == op.reservation
}

func (s *Store) runComposerQueueSteer(ctx context.Context, op *composerQueueSteer) {
	defer func() {
		s.mu.Lock()
		if s.composerQueueSteerCurrentLocked(context.Background(), op) {
			s.queueMutationScope, s.queueMutationID = "", ""
		}
		s.mu.Unlock()
		s.fire()
	}()
	for _, id := range op.ids {
		s.mu.Lock()
		current := s.composerQueueSteerCurrentLocked(ctx, op)
		pending := false
		if current {
			for _, item := range s.planBoard.Queue {
				if item.ID == id {
					pending = true
					break
				}
			}
		}
		s.mu.Unlock()
		if !current {
			return
		}
		if !pending {
			continue
		}
		action, _ := queueActionPayload(QueueActionSteer, nil)
		req := map[string]any{"sessionId": op.sessionID, "itemId": id, "action": action}
		if err := s.runUpdateQueue(ctx, op.client, op.sessionID, op.follow, id, QueueActionSteer, req, true); err != nil {
			return
		}
	}
}
