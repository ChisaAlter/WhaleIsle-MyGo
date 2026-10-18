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

type roomDeleteTransport func(*http.Request) (*http.Response, error)

func (f roomDeleteTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func roomDeleteFixturePlan(t *testing.T) DshbotBotDeletePlan {
	item := func(id, kind string, members []string, inbox []any) DshbotBotItem {
		raw, _ := json.Marshal(map[string]any{"id": id, "kind": kind, "sessionId": "session-" + id, "name": "name-" + id, "hidden": false, "memberBotIds": members, "allowedSenderIds": []string{}, "inbox": inbox})
		var p map[string]json.RawMessage
		json.Unmarshal(raw, &p)
		return DshbotBotItem{DshbotCatalogItem: DshbotCatalogItem{ID: id, Kind: kind, SessionID: "session-" + id, Name: "name-" + id, MemberBotIDs: members}, Profile: p}
	}
	row := func(raw string) map[string]json.RawMessage {
		var p map[string]json.RawMessage
		json.Unmarshal([]byte(raw), &p)
		return p
	}
	c := DshbotDeleteCatalog{Revision: 11, Items: []DshbotBotItem{item("room", "room", []string{"a", "b"}, []any{map[string]any{"id": "own-inbox"}}), item("a", "bot", []string{}, []any{map[string]any{"id": "room-task-mail", "taskId": "room-task"}, map[string]any{"id": "keep-bot-task", "taskId": "bot-task"}}), item("b", "bot", []string{}, []any{})}, Routines: []map[string]json.RawMessage{row(`{"id":"room-routine","botId":"room","name":"Room routine","prompt":"full room routine prompt"}`), row(`{"id":"bot-routine","botId":"a","name":"Member routine"}`)}, Tasks: []map[string]json.RawMessage{row(`{"id":"room-task","fromId":"room","toId":"a","status":"queued","events":[],"prompt":"full room task prompt"}`), row(`{"id":"bot-task","fromId":"a","toId":"b","status":"delivered","events":[]}`)}, Audit: []map[string]json.RawMessage{}}
	v := DshbotDeletePreview{Item: DshbotDeleteIdentity{ID: "room", Name: "name-room", Kind: "room"}, Dependencies: DshbotDeleteDependencies{Groups: []DshbotDeleteGroup{}, Routines: []DshbotDeleteNamed{{ID: "room-routine", Name: "Room routine"}}, Tasks: []DshbotDeleteTask{{ID: "room-task", Status: "queued"}}, Sources: []DshbotDeleteNamed{}}, Preserves: []string{"conversation-history", "memory"}}
	plan, e := BuildDshbotBotDeletePlan(c, v, "room")
	if e != nil {
		t.Fatal(e)
	}
	return plan
}

func TestRoomDeleteExactTargetAndPreservedMembers(t *testing.T) {
	p := roomDeleteFixturePlan(t)
	var impact map[string]json.RawMessage
	json.Unmarshal(p.Impact, &impact)
	for _, s := range []string{"session-a", "session-b", "full room routine prompt", "full room task prompt", "room-task-mail", "own-inbox", "hidden and intact", "abort active member turns"} {
		if !strings.Contains(string(p.Impact), s) {
			t.Fatal("missing room effect", s, string(p.Impact))
		}
	}
	for _, k := range []string{"groups", "peerSourcePoliciesChanged"} {
		if string(impact[k]) != "[]" {
			t.Fatal("room cascaded into another profile", k, string(impact[k]))
		}
	}
	var routines []map[string]json.RawMessage
	json.Unmarshal(impact["routineCatalogEntriesRemoved"], &routines)
	if len(routines) != 1 || string(routines[0]["id"]) != `"room-routine"` {
		t.Fatal("member routine removed", routines)
	}
	var releases []map[string]json.RawMessage
	json.Unmarshal(impact["sessionPresentationsReleased"], &releases)
	if len(releases) != 1 || string(releases[0]["itemId"]) != `"room"` {
		t.Fatal("member canonical presentation released", releases)
	}
	cmd := DshbotBotDeleteCommand{Plan: p, ConfirmedID: "room", ConfirmedName: "name-room", Acknowledged: true}
	calls := 0
	lost := false
	client := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: roomDeleteTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req struct {
			RPCID, Method string
			Payload       map[string]json.RawMessage
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "bot/delete" || len(req.Payload) != 3 || string(req.Payload["id"]) != `"room"` || string(req.Payload["revision"]) != "11" || string(req.Payload["confirmCascade"]) != "true" {
			t.Fatal("wrong target/action", req)
		}
		if lost {
			return nil, fmt.Errorf("unknown after group cancellation")
		}
		items := []any{}
		for _, i := range p.Catalog.Items {
			if i.ID != "room" {
				items = append(items, i.Profile)
			}
		}
		value := map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 12, "value": map[string]any{"items": items, "tasks": []any{}, "routines": []any{}, "audit": []any{}}}, "removed": "room", "preservedHistory": true, "preservedMemory": true, "warnings": []string{"group stop partial failure"}}
		b, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b)))}, nil
	})}}
	out, e := client.DshbotDeleteBot(context.Background(), cmd)
	if e != nil || calls != 1 || out.Removed != "room" || len(out.Warnings) != 1 {
		t.Fatal(out, e, calls)
	}
	missing := out
	missing.Catalog.Items = missing.Catalog.Items[:1]
	if ValidateDshbotBotDeleteReceipt(cmd, missing) == nil {
		t.Fatal("missing member Bot accepted")
	}
	rebound := out
	rebound.Catalog = CloneDshbotDeleteCatalog(out.Catalog)
	rebound.Catalog.Items[0].SessionID = "replacement"
	if ValidateDshbotBotDeleteReceipt(cmd, rebound) == nil {
		t.Fatal("member session rebound")
	}
	lost = true
	out, e = client.DshbotDeleteBot(context.Background(), cmd)
	if e == nil || out.Rejected || calls != 2 {
		t.Fatal("unknown room deletion retried/unlocked")
	}
	bad := CloneDshbotBotDeletePlan(p)
	bad.Catalog.Items[1].Profile["allowedSenderIds"] = json.RawMessage(`["room"]`)
	bad.Preview.Dependencies.Sources = []DshbotDeleteNamed{{ID: "a", Name: "name-a"}}
	if _, e = BuildDshbotBotDeletePlan(bad.Catalog, bad.Preview, "room"); e == nil {
		t.Fatal("unexpected peer cascade accepted")
	}
	bad = CloneDshbotBotDeletePlan(p)
	nested := CloneDshbotBotItem(p.Catalog.Items[0])
	nested.ID, nested.Name, nested.SessionID = "nested", "Nested room", "session-nested"
	nested.Profile["id"] = json.RawMessage(`"nested"`)
	nested.Profile["name"] = json.RawMessage(`"Nested room"`)
	nested.Profile["sessionId"] = json.RawMessage(`"session-nested"`)
	nested.Profile["memberBotIds"] = json.RawMessage(`["room","a"]`)
	bad.Catalog.Items = append(bad.Catalog.Items, nested)
	bad.Preview.Dependencies.Groups = []DshbotDeleteGroup{{ID: "nested", Name: "Nested room", Removed: true}}
	if _, e = BuildDshbotBotDeletePlan(bad.Catalog, bad.Preview, "room"); e == nil {
		t.Fatal("unexpected nested room cascade accepted")
	}
	bad = CloneDshbotBotDeletePlan(p)
	bad.Catalog.Items[0].Profile["memberBotIds"] = json.RawMessage(`["a","missing"]`)
	if _, e = BuildDshbotBotDeletePlan(bad.Catalog, bad.Preview, "room"); e == nil {
		t.Fatal("unresolved member preservation claimed")
	}
	obs := DshbotBotDeleteObservation(p, DshbotDeleteCatalog{Revision: 12, Items: p.Catalog.Items[1:]})
	if !strings.Contains(obs, "roomMemberBotCatalogState") || !strings.Contains(obs, "session-b") || !strings.Contains(obs, "unknown") {
		t.Fatal(obs)
	}
}
