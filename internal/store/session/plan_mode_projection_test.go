package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"whaleisle/internal/dsh"
)

func TestPlanModeProjectionEffectiveTargetAndScope(t *testing.T) {
	s := New()
	s.selectedID = "a"
	for i, tc := range []struct{ active, pending, target bool }{{false, false, false}, {false, true, true}, {true, false, true}, {true, true, false}} {
		raw, _ := json.Marshal(map[string]any{"plan": map[string]bool{"active": tc.active, "pending": tc.pending}})
		s.applyPlanModeValuesLocked(raw, float64(i+1))
		got := s.Snapshot().PlanModeControl
		if !got.Present || got.Effective != tc.target {
			t.Fatalf("%+v => %+v", tc, got)
		}
	}
	s.applyPlanModeControlLocked(json.RawMessage(`{"type":"projection","sessionId":"other","key":"plan","seq":99,"value":{"active":true,"pending":false}}`))
	s.applyPlanModeValuesLocked(json.RawMessage(`{"plan":{"active":true,"pending":false}}`), 1)
	if s.Snapshot().PlanModeControl.Effective {
		t.Fatal("stale/foreign projection won")
	}
	s.selectSessionLocked("b")
	if s.Snapshot().PlanModeControl.Present {
		t.Fatal("plan state crossed sessions")
	}
}
func TestPlanModeExitCapturedCommandAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					RPCID   any `json:"rpcId"`
					Payload struct {
						Args map[string]json.RawMessage `json:"args"`
					} `json:"payload"`
				}
				json.NewDecoder(r.Body).Decode(&body)
				calls.Add(1)
				if string(body.Payload.Args["agentId"]) != `"a"` || string(body.Payload.Args["line"]) != `"/plan off"` || string(body.Payload.Args["submittedAttachments"]) != `[]` {
					t.Errorf("wrong command %+v", body.Payload.Args)
				}
				kind := "success"
				if fail {
					kind = "error"
				}
				rpcOK(w, body.RPCID, map[string]any{"result": map[string]any{"kind": kind, "text": "refused"}})
			}))
			defer server.Close()
			s := New()
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
			s.selectedID = "a"
			s.applyPlanModeValuesLocked(json.RawMessage(`{"plan":{"active":true,"pending":false}}`), 1)
			s.ExitPlanMode(context.Background(), "other")
			if calls.Load() != 0 {
				t.Fatal("foreign target sent")
			}
			s.ExitPlanMode(context.Background(), "a")
			permissionFixtureWait(t, func() bool { return !s.Snapshot().PlanModeControl.Leaving })
			got := s.Snapshot().PlanModeControl
			if calls.Load() != 1 || (got.Error != "") != fail || !got.Effective {
				t.Fatalf("command fabricated projection or lost failure %+v calls %d", got, calls.Load())
			}
		})
	}
}
