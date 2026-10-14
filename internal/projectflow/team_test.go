package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNativeTeamAuthorityAndControls(t *testing.T) {
	calls := []string{}
	missing := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type, Method, RPCID string
			Payload             map[string]any
		}
		json.NewDecoder(r.Body).Decode(&req)
		calls = append(calls, r.URL.Path)
		var value any
		switch r.URL.Path {
		case "/dsh-project/detail":
			value = map[string]any{"project": map[string]any{"id": "p", "coordinatorSessionId": "lead", "lifecycle": "ready", "activity": map[string]any{"state": "idle"}}, "workstreams": []any{map[string]any{"id": "w", "workerSessionId": "member", "status": "running"}}, "workers": []any{map[string]any{"sessionId": "member", "phase": "idle"}}}
		case "/api/session/projections":
			args := req.Payload["args"].(map[string]any)
			request := args["request"].(map[string]any)
			if request["sessionId"] != "lead" {
				t.Error("projection must belong to Lead")
			}
			values := map[string]any{}
			if !missing {
				values["agentTeam"] = map[string]any{"members": []any{map[string]any{"id": "lead", "role": "lead", "phase": "active"}, map[string]any{"id": "member", "name": "worker", "role": "teammate", "phase": "active"}}, "tasks": []any{map[string]any{"id": "w", "ownerName": "worker", "status": "in_progress", "subject": "Build"}}}
			}
			value = map[string]any{"values": values}
		case "/dsh-project/stop":
			if req.Payload["projectId"] != "p" || req.Payload["workstreamId"] != "w" {
				t.Error("wrong stop target")
			}
			value = map[string]any{"state": "stopping"}
		case "/dsh-project/resume":
			value = map[string]any{"note": "Ready for a new user message; old requests are not automatically started."}
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL}
	ctx := context.Background()
	overview, err := c.TeamOverview(ctx, "p")
	if err != nil {
		t.Fatal(err)
	}
	if overview.MemberState(overview.Team.Members[0]) != "ready" {
		t.Fatal("Lead active phase was confused with running")
	}
	if overview.MemberState(overview.Team.Members[1]) != "assigned" {
		t.Fatal("idle member was confused with completed", overview.MemberState(overview.Team.Members[1]))
	}
	if _, err = c.Control(ctx, "p", "stop", "w"); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Control(ctx, "p", "resume", ""); err != nil {
		t.Fatal(err)
	}
	count := len(calls)
	if _, err = c.Control(ctx, "p", "complete", "w"); err == nil || len(calls) != count {
		t.Fatal("invented task CRUD permitted")
	}
	missing = true
	if _, err = c.TeamOverview(ctx, "p"); err == nil {
		t.Fatal("missing native Team projection was hidden")
	}
}
