package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParityDirectoryPreflightUsesPublishedContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type, Method, RPCID string
			Payload             map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Method != "directory/check" || req.Payload["workingDirectory"] != "/selected" || r.URL.Path != "/dsh-project/directory/check" {
			t.Error("preflight invented a desktop API", req, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"allowed": false, "reason": "home"}}})
	}))
	defer srv.Close()
	check, err := (Client{Origin: srv.URL}).CheckDirectory(context.Background(), "/selected")
	if err != nil || check.Allowed || check.Reason != "home" {
		t.Fatal(check, err)
	}
}

func TestParityRequirementVersionsAndCurrentTeam(t *testing.T) {
	var card RequestCard
	if err := json.Unmarshal([]byte(`{"versions":[{"revision":1,"goal":"old","constraints":["scope"],"conditions":["tests"],"at":123}]}`), &card); err != nil {
		t.Fatal(err)
	}
	if len(card.Versions) != 1 || card.Versions[0].Constraints[0] != "scope" || card.Versions[0].Conditions[0] != "tests" || card.Versions[0].At != 123 {
		t.Fatal("historical requirement fields were lost", card.Versions)
	}
	o := Overview{Team: &TeamProjection{Members: []TeamMember{{ID: "lead", Role: "lead"}, {ID: "active", Name: "current"}, {ID: "old", Name: "retired"}, {ID: "job", Name: "background"}}, Tasks: []TeamTask{{ID: "now", OwnerName: "current"}, {ID: "then", OwnerName: "retired", Status: "completed"}}}}
	o.Project.CurrentTaskIDs = []string{"now"}
	o.Project.Activity.RunningJobSessionIDs = []string{"job"}
	members, tasks := o.CurrentTeam(false)
	if len(members) != 3 || len(tasks) != 1 || tasks[0].ID != "now" {
		t.Fatal("current requirement/job scope not preserved", members, tasks)
	}
	members, tasks = o.CurrentTeam(true)
	if len(members) != 4 || len(tasks) != 2 || tasks[0].ID != "then" {
		t.Fatal("history does not retain original roster/tasks", members, tasks)
	}
}

func TestParityTerminalOnlyHistoricalIdentity(t *testing.T) {
	target := HistoricalTarget{Terminal: true, ProjectID: "p", WorkstreamID: "w", DelegationRef: "d", RunID: "run", RequestID: "req", RequestRevision: 2}
	if err := target.validate(); err != nil {
		t.Fatal("terminal-only record incorrectly needs a report", err)
	}
	binding := HistoricalBinding{Target: target, CoordinatorSessionID: "lead", WorkerSessionID: "member", Materialized: true, Delegation: HistoricalDelegation{Ref: "d", RunID: "run", MessageID: "message", RequestID: "req", RequestRevision: 2}}
	if _, err := binding.Address(); err != nil {
		t.Fatal(err)
	}
	binding.Delegation.RequestRevision = 3
	if _, err := binding.Address(); err == nil {
		t.Fatal("settlement followed a different requirement revision")
	}
	foreign := target
	foreign.RequestRevision = 3
	if sameHistoricalTarget(target, foreign) {
		t.Fatal("terminal source comparison omitted the frozen requirement identity")
	}
}
