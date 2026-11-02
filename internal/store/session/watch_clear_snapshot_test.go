package session

import (
	"context"
	"testing"

	"whaleisle/internal/dsh"
)

// A restarted watch must remove the previous owner's controls even while the
// new stream has not emitted a frame (or the dial has already been cancelled).
func TestWatchRestartClearsCachedControlAndBand(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(*Store, context.Context)
	}{
		{"band", func(s *Store, ctx context.Context) { s.WatchBand(ctx) }},
		{"control", func(s *Store, ctx context.Context) { s.WatchControl(ctx) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			calls := 0
			s.mu.Lock()
			s.selectedID = "session-a"
			s.mux = &dsh.Mux{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
			s.notify = func() { calls++ }
			if tc.name == "band" {
				s.bandActions = []BandAction{{ID: "old", Label: "Old action"}}
				s.bandGeneration = 5
			} else {
				s.controlNote = "old session"
			}
			s.mu.Unlock()
			before := s.Snapshot()
			if tc.name == "band" && len(before.BandActions) != 1 {
				t.Fatal("fixture lost the old band")
			}
			if tc.name == "control" && before.ControlNote != "old session" {
				t.Fatal("fixture lost the old control note")
			}
			tc.start(s, ctx)
			after := s.Snapshot()
			if tc.name == "band" && (len(after.BandActions) != 0 || after.BandGeneration != 0) {
				t.Fatalf("band survived watch restart: %+v", after.BandActions)
			}
			if tc.name == "control" && after.ControlNote != "" {
				t.Fatalf("control note survived watch restart: %q", after.ControlNote)
			}
			if calls != 1 {
				t.Fatalf("changed watch state notified %d times, want 1", calls)
			}
			tc.start(s, ctx)
			if calls != 1 {
				t.Fatalf("unchanged watch state notified again: %d", calls)
			}
		})
	}
}
