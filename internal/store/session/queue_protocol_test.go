package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"whaleisle/internal/dsh"
)

// One core fixture covers the real request and its acceptance boundaries.
func TestQueueActionProtocolAndReceipt(t *testing.T) {
	for _, tc := range []struct {
		name            string
		action          UpdateQueueAction
		accepted, stale bool
	}{
		{"remove", QueueActionRemove, true, false},
		{"steer", QueueActionSteer, true, false},
		{"rejected", QueueActionRemove, false, false},
		{"late", QueueActionRemove, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					RPCID   any `json:"rpcId"`
					Payload struct {
						Args struct {
							Request struct {
								SessionID string         `json:"sessionId"`
								ItemID    string         `json:"itemId"`
								Action    map[string]any `json:"action"`
							} `json:"request"`
						} `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				req := body.Payload.Args.Request
				if r.URL.Path != "/api/session/updateQueue" || req.SessionID != "A" || req.ItemID != "q" || req.Action["kind"] != string(tc.action) || len(req.Action) != 1 {
					t.Errorf("wrong request: %+v", req)
				}
				if tc.stale {
					s.mu.Lock()
					s.selectedID = "B"
					s.followGeneration++
					s.mu.Unlock()
				}
				rpcOK(w, body.RPCID, map[string]any{"accepted": tc.accepted})
			}))
			defer srv.Close()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			s.Attach(client, nil)
			s.selectedID = "A"
			s.planBoard.Queue = []PlanQueueItem{{ID: "q", Preview: "keep until accepted"}, {ID: "other"}}
			action, ok := queueActionPayload(tc.action, nil)
			if !ok {
				t.Fatal("valid action rejected")
			}
			s.runUpdateQueue(context.Background(), client, "A", s.followGeneration, "q", tc.action, map[string]any{"sessionId": "A", "itemId": "q", "action": action})
			snap := s.Snapshot()
			want := 2
			if tc.accepted && !tc.stale {
				want = 1
			}
			if len(snap.PlanBoard.Queue) != want {
				t.Fatalf("queue=%+v", snap.PlanBoard.Queue)
			}
			if tc.stale && snap.LastAction != "" {
				t.Fatalf("stale result changed status: %q", snap.LastAction)
			}
			if !tc.accepted && snap.LastAction != "queue: mutation was not accepted" {
				t.Fatalf("rejection=%q", snap.LastAction)
			}
		})
	}
	if _, ok := queueActionPayload(QueueActionEdit, []string{"  "}); ok {
		t.Fatal("blank edit allowed")
	}
	payload, ok := queueActionPayload(QueueActionEdit, []string{"full\ntext"})
	if !ok || payload["kind"] != "edit" || payload["content"].([]map[string]string)[0]["text"] != "full\ntext" {
		t.Fatalf("edit protocol=%v", payload)
	}
}
