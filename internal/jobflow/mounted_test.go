package jobflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMountedRosterOutlivesPopupAndRemountFencesOldStream(t *testing.T) {
	f := &fixtureTransport{opened: make(chan fixtureOpen, 8), killed: make(chan map[string]any, 8)}
	c := New(nil)
	c.Bind(f, "viewer")
	defer c.Unmount()
	c.Mount()
	c.Mount()
	roster := nextOpen(t, f)
	if roster.method != "job/list" || roster.request["sessionId"] != "viewer" {
		t.Fatal(roster)
	}
	select {
	case <-f.opened:
		t.Fatal("mount opened duplicate roster")
	default:
	}
	roster.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"live","status":"running"},{"id":"done","status":"completed"}]}`)
	s := waitState(t, c, func(s Snapshot) bool { return len(s.Rows) == 2 })
	if s.Open || !s.Mounted {
		t.Fatal("roster requires an open popup", s)
	}
	c.Open(s.Generation)
	c.ToggleSettled(s.Generation)
	c.ClearSettled(s.Generation)
	c.Close(s.Generation)
	closed := c.Snapshot()
	if !closed.Mounted || closed.Open || len(closed.Rows) != 1 || closed.SettledOpen == nil || !*closed.SettledOpen {
		t.Fatalf("close discarded mounted state: %+v", closed)
	}
	roster.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"live","status":"stopping"},{"id":"done","status":"completed"}]}`)
	waitState(t, c, func(s Snapshot) bool { return len(s.Rows) == 1 && s.Rows[0].Status == "stopping" })
	c.Open(closed.Generation)
	select {
	case <-f.opened:
		t.Fatal("popup reopen replaced roster")
	default:
	}
	c.Unmount()
	if s := c.Snapshot(); s.Mounted || len(s.Rows) != 0 || s.Open {
		t.Fatal("unmount retained watch state", s)
	}
	c.Mount()
	newRoster := nextOpen(t, f)
	roster.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"late","status":"running"}]}`)
	newRoster.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"current","status":"running"}]}`)
	waitState(t, c, func(s Snapshot) bool { return len(s.Rows) == 1 && s.Rows[0].ID == "current" })
}

type blockingKill struct {
	*fixtureTransport
	release chan struct{}
}

func (f *blockingKill) Kill(ctx context.Context, request map[string]any) (string, error) {
	f.killed <- request
	select {
	case <-f.release:
		return "", errors.New("isolated stop failure")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestPendingKillSurvivesPopupCloseWithoutDuplicate(t *testing.T) {
	f := &blockingKill{fixtureTransport: &fixtureTransport{opened: make(chan fixtureOpen, 8), killed: make(chan map[string]any, 8)}, release: make(chan struct{})}
	c := New(nil)
	c.Bind(f, "viewer")
	defer c.Unmount()
	c.Mount()
	r := nextOpen(t, f.fixtureTransport)
	r.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"live","status":"running"}]}`)
	s := waitState(t, c, func(s Snapshot) bool { return len(s.Rows) == 1 })
	c.Open(s.Generation)
	c.ArmKill(s.Generation, "live")
	c.ConfirmKill(s.Generation, "live")
	select {
	case <-f.killed:
	case <-time.After(time.Second):
		t.Fatal("stop not requested")
	}
	c.Close(s.Generation)
	g := c.Snapshot().Generation
	c.Open(g)
	c.ArmKill(g, "live")
	c.ConfirmKill(g, "live")
	select {
	case <-f.killed:
		t.Fatal("close/reopen admitted duplicate stop")
	default:
	}
	close(f.release)
	waitState(t, c, func(s Snapshot) bool { return s.Kill.Phase == "failed" && s.Kill.Error == "isolated stop failure" })
}

func TestMountedReadFailureCanRetryBeforeOpening(t *testing.T) {
	f := &fixtureTransport{opened: make(chan fixtureOpen, 8), killed: make(chan map[string]any, 8)}
	c := New(nil)
	c.Bind(f, "viewer")
	defer c.Unmount()
	c.Mount()
	r := nextOpen(t, f)
	r.stream.frames <- json.RawMessage(`{"type":"not-rows"}`)
	s := waitState(t, c, func(s Snapshot) bool { return s.Error != "" })
	if s.Open || !s.Mounted {
		t.Fatal(s)
	}
	c.Retry(s.Generation)
	retry := nextOpen(t, f)
	retry.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[]}`)
	waitState(t, c, func(s Snapshot) bool { return !s.Loading && s.Error == "" })
}
