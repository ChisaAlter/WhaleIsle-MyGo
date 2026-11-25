package session

import (
	"context"
	"sync"
	"time"

	"whaleisle/internal/errcopy"
)

const referenceQueryDebounce = 130 * time.Millisecond

// ReferenceQueryController owns debounced background reference discovery for a
// draft so UI composer views never spawn goroutines or timers in the build path.
// Within the same host/session scope, previous candidate rows are retained while
// the replacement query is in flight so keystrokes never flash the popup empty.
type ReferenceQueryController struct {
	mu      sync.Mutex
	scope   string
	key     string
	epoch   uint64
	cancel  context.CancelFunc
	pending bool
	err     string
	rows    []ReferenceCandidate
}

// Cancel revokes any in-flight reference query and clears retained rows.
func (c *ReferenceQueryController) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.key != "" || c.scope != "" {
		c.epoch++
		c.scope = ""
		c.key = ""
		c.rows = nil
		c.pending = false
		c.err = ""
	}
}

// Query returns the current candidate snapshot and, when scope/query changes,
// schedules a debounced background lookup invoking onUpdate upon completion.
func (c *ReferenceQueryController) Query(
	scope, q string,
	quoted bool,
	run func(context.Context, string, bool) ([]ReferenceCandidate, error),
	onUpdate func(),
) []ReferenceCandidate {
	key := scope + "\x00" + q
	if quoted {
		key += "\x00quoted"
	}
	c.mu.Lock()
	if c.key != key {
		if c.cancel != nil {
			c.cancel()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if c.scope != scope {
			// Switching host or session clears stale rows from the prior scope.
			c.rows = nil
		}
		c.scope = scope
		c.key = key
		c.epoch++
		epoch := c.epoch
		c.cancel = cancel
		c.pending = true
		c.err = ""
		go func() {
			timer := time.NewTimer(referenceQueryDebounce)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			rows, err := run(ctx, q, quoted)
			cancel()
			c.mu.Lock()
			if c.epoch != epoch || c.key != key {
				c.mu.Unlock()
				return
			}
			c.pending = false
			c.cancel = nil
			c.rows = append([]ReferenceCandidate(nil), rows...)
			if err != nil {
				c.err = errcopy.Sentence(err)
			}
			c.mu.Unlock()
			if onUpdate != nil {
				onUpdate()
			}
		}()
	}
	rows := append([]ReferenceCandidate(nil), c.rows...)
	c.mu.Unlock()
	return rows
}

// Status reports whether a reference query is pending and any localized error.
func (c *ReferenceQueryController) Status() (pending bool, err string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pending, c.err
}
