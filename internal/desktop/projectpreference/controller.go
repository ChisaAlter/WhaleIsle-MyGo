// Package projectpreference owns the desktop Project switch transaction.
// The Host and project data stay with the existing runtime and session owners.
package projectpreference

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrBusy = errors.New("Project preference change is pending")
var ErrRestartCancelled = errors.New("Harness restart was cancelled; Project setting was kept")

type View struct {
	Enabled          bool
	Busy             bool
	Error            string
	RestartCancelled bool
}

// Transition supplies only typed, existing desktop lifecycle operations. Protect
// must call commit only after task protection accepts this exact owned runtime.
// Write compares the current field to expected; it must preserve every other key.
type Transition struct {
	Read      func() bool
	Write     func(expected, next bool) error
	Acquire   func() (release func(), err error)
	LeaveLead func(context.Context) error
	Protect   func(context.Context, func() error) (bool, error)
	Restart   func(context.Context) error
	Notify    func()
}

type Controller struct {
	mu   sync.Mutex
	view View
}

func (c *Controller) Snapshot(accepted bool) View {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.view
	if !v.Busy {
		v.Enabled = accepted
	}
	return v
}

// Set is synchronous for its caller; the native callback runs it off the UI
// owner. Busy is published before leave/confirmation/write/restart and remains
// set until the accepted runtime has become ready or the old setting is restored.
func (c *Controller) Set(ctx context.Context, next bool, t Transition) (err error) {
	if t.Read == nil || t.Write == nil || t.Protect == nil || t.Restart == nil {
		return errors.New("Project lifecycle is unavailable")
	}
	c.mu.Lock()
	if c.view.Busy {
		c.mu.Unlock()
		return ErrBusy
	}
	previous := t.Read()
	c.view = View{Enabled: next, Busy: true}
	c.mu.Unlock()
	notify := func() {
		if t.Notify != nil {
			t.Notify()
		}
	}
	notify()
	defer func() {
		c.mu.Lock()
		c.view = View{Enabled: t.Read(), Error: errorText(err), RestartCancelled: errors.Is(err, ErrRestartCancelled)}
		c.mu.Unlock()
		notify()
	}()
	if previous == next {
		return nil
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if t.Acquire != nil {
		release, e := t.Acquire()
		if e != nil {
			return e
		}
		if release != nil {
			defer release()
		}
	}
	// This is navigation, not deletion or stopping work. Failure must abort before
	// a write, so the running Lead never disappears behind a disabled overlay.
	if !next && t.LeaveLead != nil {
		if err = t.LeaveLead(ctx); err != nil {
			return err
		}
	}
	wrote := false
	accepted, e := t.Protect(ctx, func() error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := t.Write(previous, next); e != nil {
			return e
		}
		wrote = true
		return t.Restart(ctx)
	})
	if e != nil {
		err = e
	} else if !accepted {
		err = ErrRestartCancelled
	}
	if err != nil && wrote {
		if rollback := t.Write(next, previous); rollback != nil {
			return fmt.Errorf("%w; Project rollback failed: %v", err, rollback)
		}
	}
	return err
}

func errorText(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
