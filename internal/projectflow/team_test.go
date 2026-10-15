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

func TestTeamOverviewPageRetainsHostCursorAndWaitingReason(t *testing.T) {
	reset := false
	foreign := false
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req struct {
			RPCID   string
			Payload map[string]any
		}
		json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/dsh-project/detail":
			if req.Payload["projectId"] != "p" || req.Payload["currentCursor"] != "anchor" {
				t.Error("refresh must retain the selected detail page", req.Payload)
			}
			var cursor any = "anchor"
			if reset {
				cursor = nil
			}
			if foreign {
				cursor = "unrequested"
			}
			value = map[string]any{
				"project":  map[string]any{"id": "p", "coordinatorSessionId": "lead"},
				"workPage": map[string]any{"currentCursor": cursor, "currentTotal": 23},
				"workstreams": []any{map[string]any{"id": "w", "workerSessionId": "member", "title": "Build",
					"delegation": map[string]any{"phase": "queued", "waitingReason": map[string]any{"kind": "directory", "workTitles": []string{"Other work"}}}}},
				"workers": []any{map[string]any{"sessionId": "member", "phase": "idle"}},
			}
		case "/api/session/projections":
			value = map[string]any{"values": map[string]any{"agentTeam": map[string]any{"members": []any{map[string]any{"id": "lead", "role": "lead"}}, "tasks": []any{}}}}
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL}
	overview, err := c.TeamOverviewPage(context.Background(), "p", "anchor")
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.CurrentCursors) != 2 || overview.CurrentCursors[1] != "anchor" {
		t.Fatal("selected page lost", overview.CurrentCursors)
	}
	waiting := overview.Workstreams[0].Delegation.WaitingReason
	if waiting == nil || waiting.Kind != "directory" || len(waiting.WorkTitles) != 1 || waiting.WorkTitles[0] != "Other work" {
		t.Fatal("Host queue reason lost", waiting)
	}
	members := overview.Members()
	if len(members) != 2 || members[1].ID != "member" || members[1].Description != "Build" || overview.MemberState(members[1]) != "queued" {
		t.Fatal("unprojected original Host worker lost", members)
	}
	overview.Team.Members = append(overview.Team.Members, members[1])
	if len(overview.Members()) != 2 {
		t.Fatal("Host worker duplicated an existing roster member")
	}
	reset = true
	overview, err = c.TeamOverviewPage(context.Background(), "p", "anchor")
	if err != nil || len(overview.CurrentCursors) != 1 || overview.CurrentCursors[0] != "" {
		t.Fatal("retired Host anchor did not return to the first page", overview, err)
	}
	foreign = true
	before := calls
	if _, err := c.TeamOverviewPage(context.Background(), "p", "anchor"); err == nil || calls != before+1 {
		t.Fatal("unrequested page was accepted or followed")
	}
}
