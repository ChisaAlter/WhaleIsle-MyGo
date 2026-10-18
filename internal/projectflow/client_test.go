package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNativeProjectProtocol(t *testing.T) {
	available := true
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req struct {
			Type, Method, RPCID string
			Payload             map[string]any
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.URL.Path != "/dsh-project/"+req.Method || req.Type != "client-request" || r.Header.Get("Cookie") != "auth=fixture" {
			t.Error("wrong authenticated Project envelope")
		}
		if _, ok := req.Payload["args"]; ok {
			t.Error("Project payload must not use api args wrapper")
		}
		var value any
		p := Project{ID: "project-fixture", Title: "Fixture", CoordinatorSessionID: "lead-fixture", CanonicalWorkingDirectory: "/authorized", Lifecycle: "ready"}
		switch req.Method {
		case "list":
			value = map[string]any{"available": available, "projects": []Project{p}, "error": "bridge unavailable"}
		case "create":
			if req.Payload["workingDirectory"] != "/authorized" || req.Payload["requestId"] == "" {
				t.Error("invalid create")
			}
			value = map[string]any{"project": p}
		case "rename":
			if req.Payload["projectId"] != p.ID || req.Payload["title"] != "Renamed" {
				t.Error("invalid rename")
			}
			p.Title = "Renamed"
			value = map[string]any{"project": p}
		case "detail":
			value = map[string]any{"project": p}
		default:
			t.Errorf("invented route %s", req.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL, Cookie: "auth=fixture"}
	ctx := context.Background()
	list, err := c.List(ctx)
	if err != nil || !list.Available || len(list.Projects) != 1 {
		t.Fatal(list, err)
	}
	if _, err = c.Create(ctx, "/unapproved", []string{"/authorized"}, "fixture"); err == nil {
		t.Fatal("unauthorized workspace allowed")
	}
	if calls != 1 {
		t.Fatal("unauthorized workspace transmitted")
	}
	p, err := c.Create(ctx, "/authorized", []string{"/authorized"}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	p, err = c.Rename(ctx, p.ID, "Renamed")
	if err != nil || p.Title != "Renamed" {
		t.Fatal(p, err)
	}
	lead, err := c.Open(ctx, p.ID)
	if err != nil || lead != "lead-fixture" {
		t.Fatal(lead, err)
	}
	available = false
	list, err = c.List(ctx)
	if err != nil || list.Available || list.Error == "" {
		t.Fatal("unavailability was hidden", list, err)
	}
}
