package hostterminal

import (
	"context"
	"errors"
	"sync"
	"time"
)

// OccurrenceClose is a one-shot, identity-bound explicit pane-close request.
// A failed request keeps the pane and its terminal identity available for retry.
type OccurrenceClose struct {
	mu   sync.Mutex
	done bool
	err  error
}

func (r *OccurrenceClose) Result() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done, r.err
}
func (r *OccurrenceClose) finish(err error) { r.mu.Lock(); r.done, r.err = true, err; r.mu.Unlock() }
func failedOccurrenceClose(message string) *OccurrenceClose {
	return &OccurrenceClose{done: true, err: errors.New(message)}
}

// BeginOccurrenceClose is only for the user's explicit resource-close gesture.
// It accepts a captured store/Host/session identity and never creates an owner.
// Hiding/minimizing, session switching and generic layout disposal do not call it.
func (s *Store) BeginOccurrenceClose(key string, expected *Store, owner uint64, session string) *OccurrenceClose {
	s.mu.Lock()
	child := s.occurrences[key]
	s.mu.Unlock()
	if child == nil || child != expected {
		return failedOccurrenceClose("Terminal pane changed; close the current pane instead.")
	}
	child.mu.Lock()
	if child.epoch != owner || child.ctx == nil || child.ctx.Err() != nil {
		child.mu.Unlock()
		return failedOccurrenceClose("Terminal Host changed; review the current terminal before closing.")
	}
	if child.paneClose != nil {
		done, _ := child.paneClose.Result()
		if !done {
			request := child.paneClose
			child.mu.Unlock()
			return request
		}
	}
	st := child.sessions[session]
	if st != nil && st.busy {
		child.mu.Unlock()
		return failedOccurrenceClose("Terminal operation is still running; wait before closing this pane.")
	}
	var target *entry
	if st != nil && st.active != "" {
		target = child.entries[terminalKey{session, st.active}]
	}
	if target != nil && target.closing && target.phase != "closed" {
		child.mu.Unlock()
		return failedOccurrenceClose("Terminal process close is still running; wait before closing this pane.")
	}
	if target != nil && target.phase != "closed" && (child.remote == nil || !child.ready) {
		child.mu.Unlock()
		return failedOccurrenceClose("Terminal Host is disconnected; reconnect before closing its process.")
	}
	request := &OccurrenceClose{}
	child.paneClose = request
	if st != nil {
		st.operation++
		st.busy = true
	}
	remote, ctx := child.remote, child.ctx
	id := ""
	if target != nil && target.phase != "closed" {
		id = target.key.id
		child.cancelRenameLocked(target)
		target.closing = true
		target.desired = false
		target.phase = "closing"
		target.error = ""
		child.stopHoldLocked(target)
	}
	child.mu.Unlock()
	child.fire()
	go func() {
		var err error
		if id != "" {
			callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err = remote.Call(callCtx, "terminal/close", map[string]any{"agentId": session, "id": id}, nil)
			cancel()
		}
		child.mu.Lock()
		if child.epoch != owner || child.ctx != ctx || ctx.Err() != nil {
			child.mu.Unlock()
			request.finish(errors.New("Terminal Host changed while closing; refresh before retrying."))
			child.fire()
			return
		}
		if err != nil {
			if st != nil {
				st.busy = false
			}
			if target != nil {
				target.closing = false
				target.phase = "failed"
				target.error = "Close outcome uncertain; retry close explicitly. " + err.Error()
			}
			child.mu.Unlock()
			request.finish(err)
			child.fire()
			return
		}
		// Retire the occurrence's remaining holds, including previously selected
		// identities, without sending close to those unrelated Host processes.
		for _, e := range child.entries {
			e.desired = false
			child.stopHoldLocked(e)
		}
		if target != nil {
			target.phase = "closed"
			target.info.State = "exited"
		}
		if child.cancel != nil {
			child.cancel()
		}
		child.visible = ""
		child.mu.Unlock()
		s.mu.Lock()
		if s.occurrences[key] == child {
			delete(s.occurrences, key)
		}
		s.mu.Unlock()
		request.finish(nil)
		s.fire()
	}()
	return request
}
