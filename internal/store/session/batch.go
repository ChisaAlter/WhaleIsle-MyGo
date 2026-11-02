package session

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"whaleisle/internal/dsh/api"
)

// SessionBatch is browser-local selection and a frozen, explicitly confirmed operation.
// Failed targets remain selected so they can be inspected, restored or retried.
type SessionBatchTarget struct{ ID, Title string }
type SessionBatchResult struct {
	SessionBatchTarget
	Error string
}
type SessionBatch struct {
	Enabled, Busy, Confirming bool
	Selected                  map[string]string
	Targets                   []SessionBatchTarget
	Results                   []SessionBatchResult
	Kind                      string
	Attempted                 int
	Token                     uint64
	generation                uint64
	cwd                       string
	scope                     SessionBatchScope
	scopeSet                  bool
}

func (s *Store) ensureBatchLocked() {
	if s.batch.generation != s.eventsGeneration || s.batch.cwd != s.cwd || s.batch.Selected == nil {
		s.batchToken++
		s.batch = SessionBatch{Selected: map[string]string{}, Token: s.batchToken, generation: s.eventsGeneration, cwd: s.cwd}
		// Every frame reads SessionBatch for its dialog state; advancing the
		// revision on those reads rebuilt the whole Snapshot each frame.
		s.invalidateViewLocked()
	}
}
func (s *Store) SessionBatch() SessionBatch {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureBatchLocked()
	out := s.batch
	out.Selected = map[string]string{}
	for k, v := range s.batch.Selected {
		out.Selected[k] = v
	}
	out.Targets = append([]SessionBatchTarget(nil), s.batch.Targets...)
	out.Results = append([]SessionBatchResult(nil), s.batch.Results...)
	return out
}

// EditSessionBatch changes choices within the current rendered browser scope.
func (s *Store) EditSessionBatch(action string, targets []SessionBatchTarget) {
	s.mu.Lock()
	s.ensureBatchLocked()
	if s.batch.Busy || s.batch.Confirming {
		s.mu.Unlock()
		return
	}
	switch action {
	case "toggle":
		s.batch.Enabled = !s.batch.Enabled
		s.batch.Selected = map[string]string{}
		s.batch.Results = nil
	case "clear":
		s.batch.Selected = map[string]string{}
	case "dismiss":
		s.batch.Results = nil
	case "select", "visible":
		if !s.batch.Enabled {
			s.mu.Unlock()
			return
		}
		for _, t := range targets {
			if strings.TrimSpace(t.ID) == "" {
				continue
			}
			if _, ok := s.batch.Selected[t.ID]; ok && action == "select" {
				delete(s.batch.Selected, t.ID)
			} else {
				s.batch.Selected[t.ID] = t.Title
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) RequestSessionBatch(kind string) {
	s.mu.Lock()
	s.ensureBatchLocked()
	if (kind != "archive" && kind != "delete") || s.batch.Busy || s.batch.Confirming || len(s.batch.Selected) == 0 || s.client == nil {
		s.mu.Unlock()
		return
	}
	s.batchToken++
	s.batch.Token = s.batchToken
	s.batch.Kind = kind
	s.batch.Targets = nil
	s.batch.Results = nil
	s.batch.Attempted = 0
	for id, title := range s.batch.Selected {
		s.batch.Targets = append(s.batch.Targets, SessionBatchTarget{id, title})
	}
	sort.Slice(s.batch.Targets, func(i, j int) bool { return s.batch.Targets[i].ID < s.batch.Targets[j].ID })
	s.batch.Confirming = true
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CancelSessionBatch(token uint64) {
	s.mu.Lock()
	s.ensureBatchLocked()
	if s.batch.Token == token && !s.batch.Busy {
		s.batch.Confirming = false
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) ConfirmSessionBatch(ctx context.Context, token uint64) {
	s.mu.Lock()
	s.ensureBatchLocked()
	if !s.batch.Confirming || s.batch.Busy || s.batch.Token != token || s.client == nil || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	s.batch.Busy = true
	s.batch.Confirming = false
	op, client := s.batch, s.client
	op.Targets = append([]SessionBatchTarget(nil), s.batch.Targets...)
	s.mu.Unlock()
	s.fire()
	go func() {
		for _, target := range op.Targets {
			s.mu.Lock()
			s.ensureBatchLocked()
			valid := s.batch.Token == token && s.client == client
			alreadyArchived := slices.Contains(s.archivedSessionIDs, target.ID)
			s.mu.Unlock()
			if !valid {
				return
			}
			var archived api.WorkspaceArchiveValue
			var deleted api.SessionDeleteValue
			var err error
			if op.Kind == "archive" || !alreadyArchived {
				req := api.WorkspaceArchiveSessionRequest{SessionId: target.ID}
				if op.Kind == "archive" {
					stop := true // Only explicit batch archive consents to stopping activity.
					req.StopActivity = &stop
				}
				err = client.Call(ctx, "workspace/archiveSession", map[string]any{"request": req}, &archived)
				s.mu.Lock()
				s.ensureBatchLocked()
				if s.batch.Token != token || s.client != client {
					s.mu.Unlock()
					return
				}
				if err == nil {
					if s.selectedID == target.ID {
						s.selectSessionLocked("")
					}
					s.archivedSessionIDs = append([]string(nil), archived.ArchivedSessionIds...)
					s.invalidateViewLocked()
				}
				s.mu.Unlock()
				if err == nil {
					s.fire()
				}
			}
			if op.Kind == "delete" && err == nil {
				// A synchronous notifier can change the Host after archiving.
				s.mu.Lock()
				s.ensureBatchLocked()
				valid = s.batch.Token == token && s.client == client
				s.mu.Unlock()
				if !valid {
					return
				}
				if err = ctx.Err(); err == nil {
					err = client.Call(ctx, "session/delete", map[string]any{"request": map[string]any{"sessionId": target.ID}}, &deleted)
				}
			}
			s.mu.Lock()
			s.ensureBatchLocked()
			if s.batch.Token != token || s.client != client {
				s.mu.Unlock()
				return
			}
			result := SessionBatchResult{SessionBatchTarget: target}
			if err != nil {
				result.Error = err.Error()
			} else {
				if op.Kind == "delete" {
					s.applyDeleteResultLocked(deleted)
				}
				delete(s.batch.Selected, target.ID)
			}
			s.batch.Results = append(s.batch.Results, result)
			s.batch.Attempted++
			s.mu.Unlock()
			s.fire()
		}
		s.mu.Lock()
		s.ensureBatchLocked()
		if s.batch.Token != token || s.client != client {
			s.mu.Unlock()
			return
		}
		// Failures stay retryable only in the browser scope that confirmed them.
		if s.batch.scope != op.scope {
			s.batch.Selected = map[string]string{}
		} else {
			for _, result := range s.batch.Results {
				if result.Error != "" {
					s.batch.Selected[result.ID] = result.Title
				}
			}
		}
		s.batch.Busy = false
		s.lastAction = fmt.Sprintf("batch %s: %d sessions processed", op.Kind, s.batch.Attempted)
		// Bind the final refresh to the same Host; cancellation admits no new RPC.
		var generation uint64
		if ctx.Err() == nil {
			_, generation = s.beginListRefreshLocked()
		}
		s.mu.Unlock()
		s.fire()
		if generation != 0 {
			go s.runRefreshList(ctx, client, generation)
		}
	}()
}
