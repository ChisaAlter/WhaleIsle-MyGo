package accountlogout

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type impactCancelClient struct {
	inner         *logoutFake
	onImpactStart func()
}

func (c *impactCancelClient) Call(ctx context.Context, method string, args any, out any) error {
	if method == "account/hasRunningAccountTasks" && c.onImpactStart != nil {
		// Return valid idle impact matching o.review.Impact while canceling ctx so
		// Confirm reaches line 237 (!f.current(o) right before account/signOut).
		c.onImpactStart()
		b, _ := json.Marshal(false)
		return json.Unmarshal(b, out)
	}
	return c.inner.Call(ctx, method, args, out)
}

func TestAccountSignOutCanceledContextClearsBusy(t *testing.T) {
	meta := map[string]any{"version": "fixture"}

	t.Run("confirm canceled right before signOut clears busy", func(t *testing.T) {
		f := &Flow{}
		inner := &logoutFake{status: "credential-stored", attemptID: "attempt-a", attemptPhase: "succeeded"}
		ctx, cancel := context.WithCancel(context.Background())
		client := &impactCancelClient{inner: inner}
		f.Bind(client)
		f.Request(context.Background(), meta)
		rev := f.Snapshot().ReviewID

		client.onImpactStart = cancel
		f.Confirm(ctx, meta, rev)

		got := f.Snapshot()
		if got.Busy {
			t.Fatalf("expected Busy=false after canceled Confirm, got %+v", got)
		}
		if !got.Open || got.Error != ErrorFailed {
			t.Fatalf("expected open review with ErrorFailed after canceled Confirm, got %+v", got)
		}
		if inner.exits() != 0 {
			t.Fatalf("canceled Confirm must not send signOut, got %d", inner.exits())
		}
		if !f.Cancel(got.ReviewID) {
			t.Fatalf("expected Cancel to succeed after canceled Confirm, state=%+v", got)
		}
	})

	t.Run("confirm canceled during signOut marks unknown and clears busy", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(context.Background(), meta)
		rev := f.Snapshot().ReviewID

		ctx, cancel := context.WithCancel(context.Background())
		c.signoutErr = errors.New("connection lost")
		c.afterSignout = func(*logoutFake) {
			cancel()
		}
		f.Confirm(ctx, meta, rev)

		got := f.Snapshot()
		if got.Busy {
			t.Fatalf("expected Busy=false after signOut+cancel, got %+v", got)
		}
		if !got.Unknown || got.Error != ErrorOutcomeUnknown {
			t.Fatalf("expected Unknown=true Error=%q, got %+v", ErrorOutcomeUnknown, got)
		}
	})

	t.Run("check with canceled context clears busy", func(t *testing.T) {
		f, c := newLogoutFixture()
		f.Request(context.Background(), meta)
		c.signoutErr = errors.New("lost")
		c.afterSignout = func(lf *logoutFake) {
			lf.readErr = errors.New("offline")
		}
		f.Confirm(context.Background(), meta, f.Snapshot().ReviewID)
		rev := f.Snapshot().ReviewID
		if !f.Snapshot().Unknown {
			t.Fatalf("expected Unknown=true before Check, got %+v", f.Snapshot())
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f.Check(ctx, meta, rev)
		got := f.Snapshot()
		if got.Busy {
			t.Fatalf("expected Busy=false after canceled Check, got %+v", got)
		}
		if !got.Unknown || got.Error != ErrorOutcomeUnknown {
			t.Fatalf("expected Unknown=true Error=%q, got %+v", ErrorOutcomeUnknown, got)
		}
	})
}
