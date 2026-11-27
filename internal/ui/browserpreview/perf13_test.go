package browserpreview

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/localservers"
)

func TestFloatingDocumentWatcherZeroUpdatesWhileValid(t *testing.T) {
	t.Run("ticker-skips-while-valid", func(t *testing.T) {
		done := make(chan struct{})
		tick := make(chan time.Time, 16)
		var valid atomic.Bool
		valid.Store(true)
		var expiredCalls atomic.Int32
		finished := make(chan struct{})

		go func() {
			defer close(finished)
			watchFloatingDocumentLoop(done, nil, tick, valid.Load, func() {
				expiredCalls.Add(1)
			})
		}()

		for i := 0; i < 10; i++ {
			tick <- time.Now()
		}
		// Give goroutine time to drain ticks; must trigger zero expired callbacks.
		time.Sleep(10 * time.Millisecond)
		if got := expiredCalls.Load(); got != 0 {
			t.Fatalf("onExpired called %d times while capability still valid, want 0", got)
		}

		// Flip capability invalid -> next tick fires exactly once and exits.
		valid.Store(false)
		tick <- time.Now()
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Fatal("watcher did not exit after capability became invalid")
		}
		if got := expiredCalls.Load(); got != 1 {
			t.Fatalf("onExpired called %d times after invalidation, want 1", got)
		}
	})

	t.Run("doc-done-channel-wakes-immediately", func(t *testing.T) {
		done := make(chan struct{})
		docDone := make(chan struct{})
		tick := make(chan time.Time)
		var expiredCalls atomic.Int32
		finished := make(chan struct{})

		go func() {
			defer close(finished)
			watchFloatingDocumentLoop(done, docDone, tick, func() bool { return true }, func() {
				expiredCalls.Add(1)
			})
		}()

		close(docDone)
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Fatal("watcher did not exit when docDone closed")
		}
		if got := expiredCalls.Load(); got != 1 {
			t.Fatalf("onExpired = %d, want 1", got)
		}
	})
}

func TestLocalDiscoveryLoopSkipsUnchangedSnapshots(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := []localservers.Discovery{
		{Servers: []localservers.Server{{URL: "http://127.0.0.1:3000", Port: 3000}}},
		{Servers: []localservers.Server{{URL: "http://127.0.0.1:3000", Port: 3000}}},
		{Servers: []localservers.Server{{URL: "http://127.0.0.1:3000", Port: 3000}}},
		{Servers: []localservers.Server{{URL: "http://127.0.0.1:3000", Port: 3000}, {URL: "http://127.0.0.1:5173", Port: 5173}}},
		{Servers: []localservers.Server{{URL: "http://127.0.0.1:3000", Port: 3000}, {URL: "http://127.0.0.1:5173", Port: 5173}}},
		{Servers: []localservers.Server{{URL: "http://127.0.0.1:3000", Port: 3000}, {URL: "http://127.0.0.1:5173", Port: 5173}}, Error: "probe failed"},
	}
	var idx atomic.Int32
	tick := make(chan time.Time)
	var updates atomic.Int32
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		runLocalDiscoveryLoop(ctx, tick, func(context.Context) localservers.Discovery {
			i := int(idx.Add(1)) - 1
			if i >= len(results) {
				cancel()
				return results[len(results)-1]
			}
			return results[i]
		}, func(localservers.Discovery) {
			updates.Add(1)
		})
	}()

	for i := 0; i < len(results); i++ {
		select {
		case tick <- time.Now():
		case <-finished:
		case <-time.After(time.Second):
			t.Fatalf("timed out sending tick %d", i)
		}
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("discovery loop did not exit on cancel")
	}

	// 6 polls with only 3 distinct states -> exactly 3 UI update callbacks (0 when unchanged).
	if got := updates.Load(); got != 3 {
		t.Fatalf("UI updates = %d across 6 polls with 3 distinct snapshots, want 3", got)
	}
}

func TestInteractiveDocumentLeasePureEventDriven(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	docDone := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		watchInteractiveDocumentLease(ctx, docDone, cancel)
	}()

	if ctx.Err() != nil {
		t.Fatal("ctx cancelled before docDone closed")
	}
	close(docDone)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("lease watcher did not exit when docDone closed")
	}
	if ctx.Err() == nil {
		t.Fatal("ctx was not cancelled when docDone closed")
	}
}
