package accountlogout

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

type logoutFake struct {
	mu                             sync.Mutex
	status, id                     string
	running                        bool
	impactErr, readErr, signoutErr error
	persist                        bool
	afterSignout                   func(*logoutFake)
	gate                           chan struct{}
	entered                        chan struct{}
	calls                          []string
}

func (f *logoutFake) Call(ctx context.Context, method string, args any, out any) error {
	f.mu.Lock()
	f.calls = append(f.calls, method)
	gate, entered := f.gate, f.entered
	f.gate, f.entered = nil, nil
	f.mu.Unlock()
	if entered != nil {
		close(entered)
	}
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var value any
	switch method {
	case "account/getState":
		if f.readErr != nil {
			return f.readErr
		}
		value = map[string]any{"status": f.status}
	case "account/getProfile":
		value = map[string]any{"status": "ready", "value": map[string]any{"id": f.id}}
	case "account/hasRunningAccountTasks":
		if f.impactErr != nil {
			return f.impactErr
		}
		value = f.running
	case "account/signOut":
		if f.afterSignout != nil {
			f.afterSignout(f)
		}
		if f.signoutErr != nil {
			return f.signoutErr
		}
		if !f.persist {
			f.status = "signed-out"
		}
		value = map[string]any{"status": f.status}
	default:
		return errors.New("unexpected RPC: " + method)
	}
	b, _ := json.Marshal(value)
	return json.Unmarshal(b, out)
}
func (f *logoutFake) exits() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, v := range f.calls {
		if v == "account/signOut" {
			n++
		}
	}
	return n
}
func newLogoutFixture() (*Flow, *logoutFake) {
	f := &Flow{}
	c := &logoutFake{status: "credential-stored", id: "account-a"}
	f.Bind(c)
	return f, c
}

// This fixture exercises the real account flow using only a fake RPC boundary.
// Removing confirmation, the busy guard, fresh identity/impact checks, or
// reconciliation after an uncertain write must make a corresponding case fail.
func TestAccountSignOutFixture(t *testing.T) {
	ctx := context.Background()
	meta := map[string]any{"version": "fixture"}
	t.Run("busy and completion notify without holding flow lock", func(t *testing.T) {
		f, _ := newLogoutFixture()
		var states []State
		f.SetNotify(func() { states = append(states, f.Snapshot()) })
		f.Request(ctx, meta)
		if len(states) != 2 || !states[0].Busy || states[1].Busy || !states[1].Open {
			t.Fatalf("notify: %+v", states)
		}
	})
	t.Run("open cancel and explicit commit", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		if s := f.Snapshot(); !s.Open || s.Busy || s.Impact != ImpactIdle || s.AccountID != "account-a" {
			t.Fatalf("review: %+v", s)
		}
		if c.exits() != 0 || !f.Cancel(f.Snapshot().ReviewID) {
			t.Fatal("opening/cancel sent signout")
		}
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 0 {
			t.Fatal("unreviewed confirm sent signout")
		}
		f.Request(ctx, meta)
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); s.Open || s.Busy || s.Error != "" || !s.Completed || c.exits() != 1 {
			t.Fatalf("commit: %+v exits=%d", s, c.exits())
		}
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 1 {
			t.Fatal("duplicate confirm resent")
		}
	})
	t.Run("task impact and fresh activity", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		c.running = true
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); !s.Open || s.Impact != ImpactRunning || s.Error != ErrorImpactChanged || c.exits() != 0 {
			t.Fatalf("new work: %+v", s)
		}
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 1 {
			t.Fatal("explicit updated review did not commit")
		}
		f, c = newLogoutFixture()
		c.impactErr = errors.New("offline")
		f.Request(ctx, meta)
		if s := f.Snapshot(); s.Impact != ImpactUnknown || !s.Open || s.Error != "" {
			t.Fatalf("unknown impact: %+v", s)
		}
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 1 {
			t.Fatal("reviewed unknown impact blocked")
		}
	})
	t.Run("busy blocks duplicate and dismiss", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		c.gate = make(chan struct{})
		c.entered = make(chan struct{})
		gate, entered := c.gate, c.entered
		done := make(chan struct{})
		go func() { f.Confirm(ctx, meta, f.Snapshot().ReviewID); close(done) }()
		<-entered
		if !f.Snapshot().Busy || f.Cancel(f.Snapshot().ReviewID) {
			t.Fatal("busy confirmation dismissed")
		}
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		f.Request(ctx, meta)
		close(gate)
		<-done
		if c.exits() != 1 {
			t.Fatalf("duplicate calls: %v", c.calls)
		}
	})
	t.Run("old frame clicks cannot consume a newer review", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		old := f.Snapshot().ReviewID
		c.running = true
		f.Confirm(ctx, meta, old)
		f.Confirm(ctx, meta, old)
		if c.exits() != 0 || f.Cancel(old) {
			t.Fatal("stale click consumed changed impact review")
		}
		current := f.Snapshot().ReviewID
		f.Cancel(current)
		f.Request(ctx, meta)
		f.Confirm(ctx, meta, current)
		if c.exits() != 0 {
			t.Fatal("dismissed review submitted reopened account")
		}
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 1 {
			t.Fatal("current confirmation did not work")
		}
	})
	t.Run("identity change invalidates review", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		c.id = "account-b"
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); s.Open || s.Error != ErrorAccountChanged || c.exits() != 0 {
			t.Fatalf("identity drift: %+v", s)
		}
		f.Request(ctx, meta)
		if f.Snapshot().AccountID != "account-b" {
			t.Fatal("new review unavailable")
		}
		f, c = newLogoutFixture()
		c.id = ""
		f.Request(ctx, meta)
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 0 || f.Snapshot().Error != ErrorIdentityUnavailable {
			t.Fatal("unidentified account admitted")
		}
	})
	t.Run("Host change cancels stale review reads", func(t *testing.T) {
		f, c := newLogoutFixture()
		c.gate = make(chan struct{})
		c.entered = make(chan struct{})
		entered := c.entered
		done := make(chan struct{})
		go func() { f.Request(ctx, meta); close(done) }()
		<-entered
		replacement := &logoutFake{status: "credential-stored", id: "account-b"}
		f.Bind(replacement)
		<-done
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 0 || replacement.exits() != 0 || f.Snapshot().Open {
			t.Fatal("stale Host review survived")
		}
	})
	t.Run("failure stays reviewed and needs explicit retry", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		c.signoutErr = errors.New("mock rejection")
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); !s.Open || s.Busy || s.Unknown || s.Error != ErrorFailed || c.exits() != 1 {
			t.Fatalf("failure: %+v", s)
		}
		c.signoutErr = nil
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 2 || !f.Snapshot().Completed {
			t.Fatal("explicit retry failed")
		}
	})
	t.Run("lost response reconciles signed out", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		c.signoutErr = errors.New("lost response")
		c.afterSignout = func(c *logoutFake) { c.status = "signed-out" }
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); s.Open || !s.Completed || c.exits() != 1 {
			t.Fatalf("resolved outcome: %+v", s)
		}
	})
	t.Run("unknown outcome allows only readback until resolved", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		c.signoutErr = errors.New("timeout")
		c.afterSignout = func(c *logoutFake) { c.readErr = errors.New("offline") }
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); !s.Open || !s.Unknown || s.Error != ErrorOutcomeUnknown || c.exits() != 1 {
			t.Fatalf("unknown: %+v", s)
		}
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		f.Check(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 1 {
			t.Fatal("uncertain write automatically retried")
		}
		c.readErr = nil
		c.afterSignout = nil
		f.Check(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); s.Unknown || s.Error != ErrorFailed || c.exits() != 1 {
			t.Fatalf("readback: %+v", s)
		}
		c.signoutErr = nil
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if c.exits() != 2 || !f.Snapshot().Completed {
			t.Fatal("fresh explicit confirmation unavailable")
		}
	})
	t.Run("successful RPC needs signed-out receipt", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(ctx, meta)
		c.persist = true
		f.Confirm(ctx, meta, f.Snapshot().ReviewID)
		if s := f.Snapshot(); s.Completed || !s.Open || s.Error != ErrorFailed || c.exits() != 1 {
			t.Fatalf("unverified success: %+v", s)
		}
	})
}
