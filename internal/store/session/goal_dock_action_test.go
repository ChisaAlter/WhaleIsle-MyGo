package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestGoalDockActionReportsFailureAndOnlyCommitsAcknowledgedState(t *testing.T) {
	calls, fail := 0, true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		calls++
		args := b["payload"].(map[string]any)["args"].(map[string]any)
		if args["agentId"] != "session" || args["ref"].(map[string]any)["revision"] != float64(3) || args["request"].(map[string]any)["objective"] != "keep this draft" {
			t.Errorf("wrong captured goal action: %v", args)
		}
		if fail {
			http.Error(w, "CAS rejected", http.StatusConflict)
			return
		}
		rpcOK(w, b["rpcId"], map[string]any{"id": "goal", "revision": 4, "objective": "keep this draft", "phase": "active"})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "session"
	s.planBoard.Goal = &PlanGoal{ID: "goal", Revision: 3, Objective: "accepted old objective", Phase: "active"}
	run := s.GoalDockAction("goal", 3)
	err := run(context.Background(), "edit", "keep this draft")
	if err == nil || !strings.Contains(err.Error(), "CAS rejected") || s.Snapshot().PlanBoard.Goal.Objective != "accepted old objective" {
		t.Fatalf("failure committed goal or lost error: %v %+v", err, s.Snapshot().PlanBoard.Goal)
	}
	fail = false
	if err := run(context.Background(), "edit", "keep this draft"); err != nil || s.Snapshot().PlanBoard.Goal.Objective != "keep this draft" || s.Snapshot().PlanBoard.Goal.Revision != 4 {
		t.Fatalf("acknowledged retry did not commit: %v %+v", err, s.Snapshot().PlanBoard.Goal)
	}
	if err := run(context.Background(), "edit", "obsolete revision"); err == nil || calls != 2 {
		t.Fatal("obsolete rendered CAS reached Host")
	}
}
