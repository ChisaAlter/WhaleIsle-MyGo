package session

import (
	"context"
	"testing"

	"whaleisle/internal/dsh"
)

// A restarted watch must remove the previous owner's controls even while the
// new stream has not emitted a frame (or the dial has already been cancelled).
func TestWatchRestartClearsCachedControl(t *testing.T) {
	s := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	s.mu.Lock()
	s.selectedID = "session-a"
	s.mux = &dsh.Mux{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	s.notify = func() { calls++ }
	s.controlNote = "old session"
	s.mu.Unlock()
	if before := s.Snapshot(); before.ControlNote != "old session" {
		t.Fatal("fixture lost the old control note")
	}
	s.WatchControl(ctx)
	if after := s.Snapshot(); after.ControlNote != "" {
		t.Fatalf("control note survived watch restart: %q", after.ControlNote)
	}
	if calls != 1 {
		t.Fatalf("changed watch state notified %d times, want 1", calls)
	}
	s.WatchControl(ctx)
	if calls != 1 {
		t.Fatalf("unchanged watch state notified again: %d", calls)
	}
}
