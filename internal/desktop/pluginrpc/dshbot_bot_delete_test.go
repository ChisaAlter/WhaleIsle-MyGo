package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type botDeleteTransport func(*http.Request) (*http.Response, error)

func (f botDeleteTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func botDeleteFixtureView(revision int) map[string]any {
	item := func(id, kind string, members, senders []string, inbox []any) map[string]any {
		return map[string]any{"id": id, "sessionId": "session-" + id, "name": "name-" + id, "title": "title-" + id, "kind": kind, "memberBotIds": members, "allowedSenderIds": senders, "inbox": inbox, "hidden": false}
	}
	return map[string]any{"ns": "dsh-bot", "revision": revision, "value": map[string]any{
		"items":    []any{item("a", "bot", []string{}, []string{}, []any{map[string]any{"id": "own-mail"}}), item("b", "bot", []string{}, []string{"a"}, []any{map[string]any{"id": "task-mail", "taskId": "t1"}, map[string]any{"id": "routine-mail", "routineId": "r/1"}, map[string]any{"id": "keep"}}), item("room-small", "room", []string{"a", "b"}, []string{}, []any{map[string]any{"id": "room-mail"}}), item("room-large", "room", []string{"a", "b", "c"}, []string{}, []any{}), item("c", "bot", []string{}, []string{}, []any{})},
		"routines": []any{map[string]any{"id": "r/1", "botId": "a", "name": "Delete routine", "prompt": "whole routine prompt"}, map[string]any{"id": "r_1", "botId": "b", "name": "Shared notepad"}},
		"tasks":    []any{map[string]any{"id": "t1", "fromId": "a", "toId": "b", "status": "delivered", "events": []any{}, "prompt": "complete task body"}, map[string]any{"id": "t-done", "fromId": "a", "toId": "b", "status": "completed", "events": []any{}}}, "audit": []any{}, "triggerToken": "must-not-retain-or-display"}}
}
func botDeleteFixturePreview() DshbotDeletePreview {
	return DshbotDeletePreview{Item: DshbotDeleteIdentity{ID: "a", Name: "title-a", Kind: "bot"}, Dependencies: DshbotDeleteDependencies{Groups: []DshbotDeleteGroup{{ID: "room-small", Name: "name-room-small", Removed: true}, {ID: "room-large", Name: "name-room-large", Removed: false}}, Routines: []DshbotDeleteNamed{{ID: "r/1", Name: "Delete routine"}}, Tasks: []DshbotDeleteTask{{ID: "t1", Status: "delivered"}}, Sources: []DshbotDeleteNamed{{ID: "b", Name: "title-b"}}}, Preserves: []string{"conversation-history", "memory"}}
}
func TestBotDeleteContractCompletePreviewAndSingleWrite(t *testing.T) {
	view := botDeleteFixtureView(7)
	raw, _ := json.Marshal(view)
	catalog, err := decodeDshbotDeleteCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildDshbotBotDeletePlan(catalog, botDeleteFixturePreview(), "a")
	if err != nil {
		t.Fatal(err)
	}
	all, _ := json.Marshal(plan)
	if strings.Contains(string(all), "must-not-retain") {
		t.Fatal("retained trigger credential")
	}
	for _, s := range []string{"whole routine prompt", "complete task body", "r_1.md", "Shared notepad", "task-mail", "own-mail", "room-mail", "session-room-small", "room-large"} {
		if !strings.Contains(string(plan.Impact), s) {
			t.Fatal("missing complete effect", s, string(plan.Impact))
		}
	}
	cmd := DshbotBotDeleteCommand{Plan: plan, ConfirmedID: "a", ConfirmedName: "name-a", Acknowledged: true}
	calls, writes := 0, 0
	drift := false
	lost := false
	wrongReceipt := false
	client := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: botDeleteTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req struct {
			RPCID, Method string
			Payload       map[string]json.RawMessage
		}
		json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch req.Method {
		case "describe":
			v := botDeleteFixtureView(7)
			if drift && calls%3 == 0 {
				v["revision"] = 8
			}
			value = map[string]any{"view": v}
		case "bot/delete-preview":
			if len(req.Payload) != 1 || string(req.Payload["id"]) != `"a"` {
				t.Fatal(req.Payload)
			}
			value = botDeleteFixturePreview()
		case "bot/delete":
			writes++
			if len(req.Payload) != 3 || string(req.Payload["id"]) != `"a"` || string(req.Payload["revision"]) != "7" || string(req.Payload["confirmCascade"]) != "true" {
				t.Fatal(req.Payload)
			}
			if lost {
				return nil, fmt.Errorf("lost after stop or file deletion")
			}
			next := botDeleteFixtureView(8)
			v := next["value"].(map[string]any)
			v["items"] = []any{}
			v["routines"] = []any{}
			v["tasks"] = []any{}
			id := "a"
			if wrongReceipt {
				id = "other"
			}
			value = map[string]any{"view": next, "removed": id, "preservedHistory": true, "preservedMemory": true, "warnings": []string{"Could not remove routine notepad r/1: complete warning"}}
		default:
			t.Fatal("unexpected operation", req.Method)
		}
		b, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b)))}, nil
	})}}
	loaded, err := client.DshbotBotDeletePlan(context.Background(), "a")
	if err != nil || !SameDshbotBotDeletePlan(plan, loaded) || calls != 3 || writes != 0 {
		t.Fatal("read contract", err, calls)
	}
	drift = true
	if _, err = client.DshbotBotDeletePlan(context.Background(), "a"); err == nil || writes != 0 {
		t.Fatal("drift accepted")
	}
	drift = false
	no := cmd
	no.Acknowledged = false
	if out, e := client.DshbotDeleteBot(context.Background(), no); e == nil || out.Attempted || writes != 0 {
		t.Fatal("danger confirmation bypass")
	}
	no = cmd
	no.ConfirmedName = "title-a"
	if _, e := client.DshbotDeleteBot(context.Background(), no); e == nil || writes != 0 {
		t.Fatal("display title substituted for exact name")
	}
	out, err := client.DshbotDeleteBot(context.Background(), cmd)
	if err != nil || out.Removed != "a" || len(out.Warnings) != 1 || writes != 1 {
		t.Fatal(out, err, writes)
	}
	lost = true
	out, err = client.DshbotDeleteBot(context.Background(), cmd)
	if err == nil || !out.Attempted || out.Rejected || writes != 2 {
		t.Fatal("unknown failure retried or unlocked")
	}
	lost = false
	wrongReceipt = true
	out, err = client.DshbotDeleteBot(context.Background(), cmd)
	if err == nil || out.Rejected || len(out.Warnings) != 1 || writes != 3 {
		t.Fatal("wrong receipt accepted or warning lost")
	}
	bad := botDeleteFixturePreview()
	bad.Dependencies.Groups = bad.Dependencies.Groups[:1]
	if _, e := BuildDshbotBotDeletePlan(catalog, bad, "a"); e == nil {
		t.Fatal("incomplete preview accepted")
	}
	observation := DshbotBotDeleteObservation(plan, DshbotDeleteCatalog{Revision: 9})
	if !strings.Contains(observation, "absent") || !strings.Contains(observation, "unknown") {
		t.Fatal(observation)
	}
}
