package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

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

type bandPressTestTransport func(*http.Request) (*http.Response, error)

func (f bandPressTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPressBandDiscardsStaleResponseAcrossSessionSwitch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered := make(chan struct{})
		release := make(chan struct{})
		var enteredOnce sync.Once
		client := &dsh.Client{
			Endpoint: dsh.Endpoint{Origin: "https://band.invalid"},
			HTTP: &http.Client{Transport: bandPressTestTransport(func(r *http.Request) (*http.Response, error) {
				var env struct {
					RPCID string `json:"rpcId"`
				}
				_ = json.NewDecoder(r.Body).Decode(&env)
				if !strings.HasSuffix(r.URL.Path, "/claudeCodeMods/pressBand") {
					raw, _ := json.Marshal(map[string]any{
						"type":   "server-response",
						"rpcId":  env.RPCID,
						"result": map[string]any{"ok": true, "value": map[string]any{}},
					})
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw))), Request: r}, nil
				}
				enteredOnce.Do(func() { close(entered) })
				<-release
				raw, _ := json.Marshal(map[string]any{
					"type":  "server-response",
					"rpcId": env.RPCID,
					"result": map[string]any{
						"ok": true,
						"value": map[string]any{
							"generation": 9,
							"tree":       []any{map[string]any{"actionId": "stale-btn", "label": "Stale"}},
						},
					},
				})
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw))), Request: r}, nil
			})},
		}

		s := New()
		s.Attach(client, nil)
		s.mu.Lock()
		s.selectedID = "session-a"
		s.bandGeneration = 3
		s.bandActions = []BandAction{{ID: "old", Label: "Old"}}
		s.mu.Unlock()

		s.PressBand(context.Background(), "old")
		<-entered
		s.Select("session-b")
		s.mu.Lock()
		s.bandGeneration = 1
		s.bandActions = []BandAction{{ID: "b-btn", Label: "Session B"}}
		s.lastAction = "kept-b"
		s.mu.Unlock()

		close(release)
		synctest.Wait()

		snap := s.Snapshot()
		if snap.SelectedID != "session-b" || snap.BandGeneration != 1 || len(snap.BandActions) != 1 || snap.BandActions[0].ID != "b-btn" || snap.LastAction != "kept-b" {
			t.Fatalf("stale PressBand mutated session-b state: %+v", snap)
		}
	})
}
