package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectLifecycleIsNotRequestArchive(t *testing.T) {
	state := "ready"
	writes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		json.NewDecoder(r.Body).Decode(&q)
		if q.Payload["projectId"] != "p" || q.Payload["requestId"] != nil {
			t.Error("wrong lifecycle identity")
		}
		note := ""
		switch q.Method {
		case "detail":
		case "archive":
			state = "archived"
			writes++
		case "restore":
			state = "ready"
			writes++
			note = "Ready for a new user message; old requests are not automatically started."
		default:
			t.Fatal("invented lifecycle route")
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"project": map[string]any{"id": "p", "lifecycle": state, "activity": map[string]any{"state": "idle", "running": 1, "queued": 2}}, "note": note}}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL}
	ctx := context.Background()
	p, err := c.LifecycleProject(ctx, "p")
	if err != nil || !p.HasActiveWork() || writes != 0 {
		t.Fatal("prepare must be read only and background work requires confirmation", p, err)
	}
	p.Activity.Running = 0
	p.Activity.Queued = 0
	if p.HasActiveWork() {
		t.Fatal("idle alone is not active work")
	}
	p.Activity.State = "stopping"
	if !p.HasActiveWork() {
		t.Fatal("stopping requires confirmation")
	}
	result, err := c.SetProjectLifecycle(ctx, "p", "archive")
	if err != nil || result.Project.Lifecycle != "archived" {
		t.Fatal(result, err)
	}
	result, err = c.SetProjectLifecycle(ctx, "p", "restore")
	if err != nil || result.Note == "" || result.Project.Lifecycle != "ready" || writes != 2 {
		t.Fatal(result, err)
	}
	if _, err = c.SetProjectLifecycle(ctx, "p", "delete"); err == nil || writes != 2 {
		t.Fatal("unsupported lifecycle mutation")
	}
}
