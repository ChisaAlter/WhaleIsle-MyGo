package mcpconfig

import (
	"context"
	"time"
)

// HealthRefreshInterval matches the original settings panel's HEALTH_POLL_MS.
const HealthRefreshInterval = 2 * time.Second

const healthReadTimeout = 15 * time.Second

type healthRefreshLoop struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

type healthRefreshRead struct {
	cancel      context.CancelFunc
	invalidated bool
}

type healthTickerFactory func() (<-chan time.Time, func())

// ObserveHealthRefresh is called only for the current Host and rendered panel.
// It never performs an initial load, connection test, retry, or authorization.
// A ready list starts a single loop; hiding the panel cancels it immediately.
func (c *Controller) ObserveHealthRefresh(visible bool, client Caller, notify func()) {
	c.observeHealthRefresh(visible, client, notify, func() (<-chan time.Time, func()) {
		ticker := time.NewTicker(HealthRefreshInterval)
		return ticker.C, ticker.Stop
	})
}

func (c *Controller) observeHealthRefresh(visible bool, client Caller, notify func(), ticker healthTickerFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !visible || client == nil || !c.state.Ready {
		c.stopHealthRefreshLocked()
		return
	}
	if c.healthLoop != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	loop := &healthRefreshLoop{ctx: ctx, cancel: cancel, done: make(chan struct{})}
	c.healthLoop = loop
	ticks, stop := ticker()
	go func() {
		defer close(loop.done)
		defer cancel()
		defer stop()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-ticks:
				if !ok {
					return
				}
			}
			if c.refreshHealth(loop, client) && notify != nil {
				notify()
			}
		}
	}()
}

// StopHealthRefresh must run before replacing this Host's controller, and on
// native window hide, minimize, or close. Late replies cannot publish afterward.
func (c *Controller) StopHealthRefresh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopHealthRefreshLocked()
}

func (c *Controller) stopHealthRefreshLocked() {
	if loop := c.healthLoop; loop != nil {
		c.healthLoop = nil
		loop.cancel()
	}
}

// Foreground operations take priority without disabling their buttons during a
// background list. Keep the in-flight slot until the canceled call returns.
func (c *Controller) cancelHealthReadLocked() {
	if read := c.healthRead; read != nil {
		read.invalidated = true
		read.cancel()
	}
}

func (c *Controller) refreshHealth(loop *healthRefreshLoop, client Caller) bool {
	c.mu.Lock()
	if c.healthLoop != loop || loop.ctx.Err() != nil || !c.state.Ready || c.state.Busy || c.healthRead != nil {
		c.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithTimeout(loop.ctx, healthReadTimeout)
	read := &healthRefreshRead{cancel: cancel}
	c.healthRead = read
	c.mu.Unlock()
	defer cancel()

	rows, err := readRows(ctx, client)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.healthRead == read {
		c.healthRead = nil
	}
	if c.healthLoop != loop || loop.ctx.Err() != nil || read.invalidated || ctx.Err() == context.Canceled || c.state.Busy {
		return false
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		// A transient read failure keeps the last snapshot and action feedback.
		c.state.HealthRefreshFailed = true
		return true
	}
	c.state.Rows = rows.Servers
	c.reconcileDiagnostics()
	c.state.HealthRefreshFailed = false
	return true
}
