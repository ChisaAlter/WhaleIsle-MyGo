package hostterminal

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Write forwards native keyboard input to the Host as one terminal/write call
// bound to this attachment. Input stays serialized by ioMu; a call that fails
// or times out has an uncertain outcome, so the attachment fails and the input
// is never replayed on recovery.
func (a *Attachment) Write(p []byte) (int, error) {
	a.ioMu.Lock()
	defer a.ioMu.Unlock()
	s := a.store
	s.mu.Lock()
	valid := s.writableLocked(a)
	maxBytes := s.sessionLocked(a.entry.key.session).environment.MaxInputBytes
	s.mu.Unlock()
	if !valid {
		return 0, errors.New("Host terminal attachment is not writable")
	}
	if len(p) > maxBytes {
		a.fail(errors.New("Host terminal input exceeds its byte limit"))
		return 0, errors.New("Host terminal input exceeds its byte limit")
	}
	if len(p) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	err := a.remote.Call(ctx, "terminal/write", map[string]any{
		"agentId":      a.entry.key.session,
		"id":           a.entry.key.id,
		"attachmentId": a.ID,
		"data":         string(p),
	}, nil)
	if err != nil {
		a.fail(fmt.Errorf("Host terminal input was not accepted; it will not be replayed: %w", err))
		return 0, err
	}
	return len(p), nil
}

// Resize proposes a new grid to the Host as one terminal/resize call bound to
// this attachment, clamped to the Host environment's limits. A read-only or
// recovering attachment keeps the Host's authoritative grid.
func (a *Attachment) Resize(cols, rows int) error {
	a.ioMu.Lock()
	defer a.ioMu.Unlock()
	s := a.store
	s.mu.Lock()
	if !s.writableLocked(a) {
		s.mu.Unlock()
		return errors.New("Host terminal attachment is not writable")
	}
	env := s.sessionLocked(a.entry.key.session).environment
	cols, rows = min(cols, env.MaxCols), min(rows, env.MaxRows)
	same := a.entry.info.Cols == cols && a.entry.info.Rows == rows
	s.mu.Unlock()
	if cols < 2 || rows < 1 || same {
		return nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	err := a.remote.Call(ctx, "terminal/resize", map[string]any{
		"agentId":      a.entry.key.session,
		"id":           a.entry.key.id,
		"attachmentId": a.ID,
		"cols":         cols,
		"rows":         rows,
	}, nil)
	if err != nil {
		a.fail(fmt.Errorf("Host terminal resize was not accepted: %w", err))
	}
	return err
}
