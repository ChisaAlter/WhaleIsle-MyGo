package main

import (
	"sync"
	"time"
)

const storeRefreshCoalesceWindow = 16 * time.Millisecond

// storeRefreshCoalescer batches high-frequency store/SSE notifications into at
// most one UI flush per window while flushing leading-edge and terminal/urgent
// transitions immediately.
type storeRefreshCoalescer struct {
	mu        sync.Mutex
	window    time.Duration
	now       func() time.Time
	onFlush   func()
	lastFlush time.Time
	pending   bool
	timer     *time.Timer
	closed    bool
}

func newStoreRefreshCoalescer(window time.Duration, onFlush func()) *storeRefreshCoalescer {
	if window <= 0 {
		window = storeRefreshCoalesceWindow
	}
	return &storeRefreshCoalescer{
		window:  window,
		now:     time.Now,
		onFlush: onFlush,
	}
}

// Notify schedules a coalesced flush. When urgent is true (e.g., terminal
// stream completion, error, or interactive approval), or when at least window
// has elapsed since the previous flush, Notify flushes immediately.
func (c *storeRefreshCoalescer) Notify(urgent bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	now := c.now()
	elapsed := time.Duration(0)
	hasPrev := !c.lastFlush.IsZero()
	if hasPrev {
		elapsed = now.Sub(c.lastFlush)
	}
	if urgent || !hasPrev || elapsed >= c.window {
		if c.timer != nil {
			c.timer.Stop()
			c.timer = nil
		}
		c.pending = false
		c.lastFlush = now
		fn := c.onFlush
		c.mu.Unlock()
		if fn != nil {
			fn()
		}
		return
	}
	c.pending = true
	if c.timer == nil {
		delay := c.window - elapsed
		if delay < time.Millisecond {
			delay = time.Millisecond
		}
		c.timer = time.AfterFunc(delay, c.flushFromTimer)
	}
	c.mu.Unlock()
}

func (c *storeRefreshCoalescer) flushFromTimer() {
	c.mu.Lock()
	if c.closed {
		c.timer = nil
		c.pending = false
		c.mu.Unlock()
		return
	}
	c.timer = nil
	if !c.pending {
		c.mu.Unlock()
		return
	}
	c.pending = false
	c.lastFlush = c.now()
	fn := c.onFlush
	c.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// Close cancels any scheduled flush timer and suppresses future notifications.
func (c *storeRefreshCoalescer) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.closed = true
	c.pending = false
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}
