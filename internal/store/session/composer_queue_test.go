package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"whaleisle/internal/dsh"
)

type composerQueueTransport func(*http.Request) (*http.Response, error)

func (f composerQueueTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestEmptyComposerQueueSteerFIFOAndOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, code                         string
		reject, replaceHost, changeSession bool
	}{
		{name: "FIFO"}, {name: "unavailable", code: "session/steer-unavailable"}, {name: "already-consumed", code: "session/queue-item-not-found"}, {name: "real-error", code: "fixture/failure"}, {name: "rejected", reject: true}, {name: "Host-replaced", replaceHost: true}, {name: "session-changed", changeSession: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			var calls []string
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}}
			client.HTTP = &http.Client{Transport: composerQueueTransport(func(r *http.Request) (*http.Response, error) {
				var body struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						Args struct {
							Request map[string]json.RawMessage `json:"request"`
						} `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				req := body.Payload.Args.Request
				var sid, id string
				var action map[string]any
				_ = json.Unmarshal(req["sessionId"], &sid)
				_ = json.Unmarshal(req["itemId"], &id)
				_ = json.Unmarshal(req["action"], &action)
				if r.URL.Path != "/api/session/updateQueue" || sid != "A" || len(req) != 3 || len(action) != 1 || action["kind"] != "steer" {
					t.Fatalf("wrong actual request: %s %s", r.URL.Path, req)
				}
				calls = append(calls, id)
				s.mu.Lock()
				if len(calls) == 1 {
					// A second chord while this explicit batch is pending cannot duplicate it.
					if again := s.beginComposerQueueSteerLocked("A", s.composerHostGeneration); again != nil {
						t.Fatal("duplicate batch admitted")
					}
					s.planBoard.Queue = append(s.planBoard.Queue, PlanQueueItem{ID: "new-after-gesture"})
				}
				s.mu.Unlock()
				if len(calls) == 1 && tc.replaceHost {
					s.Attach(&dsh.Client{}, nil)
					s.mu.Lock()
					s.lastAction = "new owner"
					s.planBoard.Queue = []PlanQueueItem{{ID: "new-owner-row"}}
					s.mu.Unlock()
				}
				if len(calls) == 1 && tc.changeSession {
					s.mu.Lock()
					s.selectSessionLocked("B")
					s.lastAction = "new owner"
					s.planBoard.Queue = []PlanQueueItem{{ID: "new-owner-row"}}
					s.mu.Unlock()
				}
				result := map[string]any{"ok": true, "value": map[string]any{"accepted": true}}
				if len(calls) == 2 {
					if tc.code != "" {
						result = map[string]any{"ok": false, "error": map[string]any{"code": tc.code, "message": "fixture error"}}
					}
					if tc.reject {
						result = map[string]any{"ok": true, "value": map[string]any{"accepted": false}}
					}
				}
				raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": result})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
			})}
			s.Attach(client, nil)
			s.selectedID = "A"
			s.turnStartMs = 1
			s.applyQueueInboxLocked(json.RawMessage(`{"next-turn":[{"id":"first","content":[{"type":"text","text":"never re-send this text"}]},{"id":"second","content":[]}]}`), 1)
			epoch := s.composerHostGeneration
			if got := s.beginComposerQueueSteerLocked("A", epoch-1); got != nil {
				t.Fatal("stale Host gesture admitted")
			}
			savedProjection := s.queueProjectionGeneration
			s.queueProjectionGeneration++
			if got := s.beginComposerQueueSteerLocked("A", epoch); got != nil {
				t.Fatal("historical/noncurrent queue admitted")
			}
			s.queueProjectionGeneration = savedProjection
			s.turnStartMs = 0
			if got := s.beginComposerQueueSteerLocked("A", epoch); got != nil {
				t.Fatal("idle queue admitted")
			}
			s.turnStartMs = 1
			op := s.beginComposerQueueSteerLocked("A", epoch)
			if op == nil || !s.queueActionsLocked().Busy {
				t.Fatal("explicit current queue not reserved")
			}
			if len(calls) != 0 {
				t.Fatal("preparation dispatched before explicit worker")
			}
			s.runComposerQueueSteer(context.Background(), op)
			if tc.replaceHost || tc.changeSession {
				if !reflect.DeepEqual(calls, []string{"first"}) || s.lastAction != "new owner" || len(s.planBoard.Queue) != 1 || s.planBoard.Queue[0].ID != "new-owner-row" {
					t.Fatal("late receipt mutated new owner", calls, s.lastAction, s.planBoard.Queue)
				}
				return
			}
			if !reflect.DeepEqual(calls, []string{"first", "second"}) {
				t.Fatal("FIFO or fixed snapshot violated", calls)
			}
			if s.queueActionsLocked().Busy {
				t.Fatal("owned reservation leaked")
			}
			want := []string{"new-after-gesture"}
			if tc.code != "" || tc.reject {
				want = []string{"second", "new-after-gesture"}
			}
			var remaining []string
			for _, item := range s.planBoard.Queue {
				remaining = append(remaining, item.ID)
			}
			if !reflect.DeepEqual(remaining, want) {
				t.Fatal("accepted rows resent or failed rows lost", remaining, want)
			}
			if tc.code == "fixture/failure" && !strings.Contains(s.lastAction, "fixture/failure") {
				t.Fatal("genuine error hidden", s.lastAction)
			}
			if tc.reject && s.lastAction != "queue: mutation was not accepted" {
				t.Fatal("rejection hidden", s.lastAction)
			}
			if tc.code != "" && tc.code != "fixture/failure" && s.lastAction != "queue steer first" {
				t.Fatal("known convergence surfaced as error", s.lastAction)
			}
		})
	}
}
