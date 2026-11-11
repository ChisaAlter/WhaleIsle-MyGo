package pluginrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestWhaleTaskNativeRPCUsesNarrowControlAndExplicitOwner(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body clientRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		calls = append(calls, body.Method)
		payload, _ := body.Payload.(map[string]any)
		var value any = map[string]any{"id": "task-1", "state": "queued", "generation": 1}
		switch body.Method {
		case "tasks/list":
			value = map[string]any{"available": true, "tasks": []any{value}}
		case "tasks/stop", "tasks/resume":
			if !reflect.DeepEqual(payload, map[string]any{"taskId": "task-1"}) {
				t.Errorf("task control changed authorization: %v", payload)
			}
		case "tasks/owner":
			if !reflect.DeepEqual(payload, map[string]any{"channel": "feishu", "botId": "bot-1", "actor": "open-id"}) {
				t.Errorf("owner=%v", payload)
			}
			value = map[string]any{"ok": true}
		case "tasks/revoke-owner":
			if !reflect.DeepEqual(payload, map[string]any{"botId": "bot-1"}) {
				t.Errorf("revoke=%v", payload)
			}
			value = map[string]any{"ok": true}
		}
		if r.URL.Path != "/dsh-whale/"+body.Method {
			t.Errorf("wrong path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL}
	ctx := context.Background()
	if _, err := c.WhaleStartTask(ctx, WhaleTaskDraft{Goal: "Summarize", Files: []string{"C:\\notes.md"}}); err != nil {
		t.Fatal(err)
	}
	if value, err := c.WhaleTasks(ctx); err != nil || !value.Available || len(value.Tasks) != 1 {
		t.Fatal(value, err)
	}
	for _, action := range []string{"stop", "resume"} {
		if _, err := c.WhaleControlTask(ctx, "task-1", action); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.WhaleTaskOwner(ctx, "bot-1", "open-id", false); err != nil {
		t.Fatal(err)
	}
	if err := c.WhaleTaskOwner(ctx, "bot-1", "", true); err != nil {
		t.Fatal(err)
	}
	if err := c.WhaleTaskOwner(ctx, "bot-1", "*", false); err == nil {
		t.Fatal("wildcard owner admitted")
	}
	if _, err := c.WhaleControlTask(ctx, "task-1", "delete"); err == nil {
		t.Fatal("unsupported control admitted")
	}
	if strings.Join(calls, ",") != "tasks/start,tasks/list,tasks/stop,tasks/resume,tasks/owner,tasks/revoke-owner" {
		t.Fatal(calls)
	}
}
