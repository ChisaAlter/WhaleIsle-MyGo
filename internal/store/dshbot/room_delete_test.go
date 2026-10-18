package dshbot

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func roomDeleteFixturePlan(t *testing.T) pluginrpc.DshbotBotDeletePlan {
	item := func(id, kind string, members []string, inbox []any) pluginrpc.DshbotBotItem {
		raw, _ := json.Marshal(map[string]any{"id": id, "kind": kind, "sessionId": "session-" + id, "name": "name-" + id, "hidden": false, "memberBotIds": members, "allowedSenderIds": []string{}, "inbox": inbox})
		var p map[string]json.RawMessage
		json.Unmarshal(raw, &p)
		return pluginrpc.DshbotBotItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: id, Kind: kind, SessionID: "session-" + id, Name: "name-" + id, MemberBotIDs: members}, Profile: p}
	}
	row := func(raw string) map[string]json.RawMessage {
		var p map[string]json.RawMessage
		json.Unmarshal([]byte(raw), &p)
		return p
	}
	c := pluginrpc.DshbotDeleteCatalog{Revision: 11, Items: []pluginrpc.DshbotBotItem{item("room", "room", []string{"a", "b"}, []any{map[string]any{"id": "own-inbox"}}), item("a", "bot", []string{}, []any{map[string]any{"id": "room-task-mail", "taskId": "room-task"}, map[string]any{"id": "keep-bot-task", "taskId": "bot-task"}}), item("b", "bot", []string{}, []any{})}, Routines: []map[string]json.RawMessage{row(`{"id":"room-routine","botId":"room","name":"Room routine","prompt":"full room routine prompt"}`), row(`{"id":"bot-routine","botId":"a","name":"Member routine"}`)}, Tasks: []map[string]json.RawMessage{row(`{"id":"room-task","fromId":"room","toId":"a","status":"queued","events":[],"prompt":"full room task prompt"}`), row(`{"id":"bot-task","fromId":"a","toId":"b","status":"delivered","events":[]}`)}, Audit: []map[string]json.RawMessage{}}
	v := pluginrpc.DshbotDeletePreview{Item: pluginrpc.DshbotDeleteIdentity{ID: "room", Name: "name-room", Kind: "room"}, Dependencies: pluginrpc.DshbotDeleteDependencies{Groups: []pluginrpc.DshbotDeleteGroup{}, Routines: []pluginrpc.DshbotDeleteNamed{{ID: "room-routine", Name: "Room routine"}}, Tasks: []pluginrpc.DshbotDeleteTask{{ID: "room-task", Status: "queued"}}, Sources: []pluginrpc.DshbotDeleteNamed{}}, Preserves: []string{"conversation-history", "memory"}}
	plan, e := pluginrpc.BuildDshbotBotDeletePlan(c, v, "room")
	if e != nil {
		t.Fatal(e)
	}
	return plan
}

func TestRoomDeleteUnknownKeepsOriginalRoomLock(t *testing.T) {
	p := roomDeleteFixturePlan(t)
	var c BotDeleter
	c.Bind("room-account", "client")
	q, ok := c.BeginLoad("room-account")
	if !ok {
		t.Fatal("read")
	}
	c.FinishLoad(q, p, pluginrpc.DshbotDeleteCatalog{}, true, nil)
	c.Prepare("room-account")
	r := c.Snapshot("room-account").Review
	c.ConfirmIdentity("room-account", r, "a", "name-a")
	c.Acknowledge("room-account", r, true)
	if _, _, ok := c.Begin("room-account", r); ok {
		t.Fatal("member substituted for reviewed room")
	}
	c.ConfirmIdentity("room-account", r, "room", "name-room")
	ticket, cmd, ok := c.Begin("room-account", r)
	if !ok || cmd.Plan.Target.Kind != "room" || !c.MarkSent(ticket) {
		t.Fatal("room review")
	}
	c.Suspend()
	c.Finish(ticket, pluginrpc.DshbotBotDeleteResult{Attempted: true}, fmt.Errorf("lost room receipt"))
	c.Bind("room-account", "replacement-client")
	q, ok = c.BeginLoad("room-account")
	if !ok {
		t.Fatal("unknown observation")
	}
	c.FinishLoad(q, pluginrpc.DshbotBotDeletePlan{}, pluginrpc.DshbotDeleteCatalog{Revision: 12, Items: p.Catalog.Items[1:]}, true, nil)
	c.Cancel("room-account")
	c.Prepare("room-account")
	s := c.Snapshot("room-account")
	if !s.Uncertain || s.Pending == nil || s.Pending.Plan.Target.ID != "room" || s.Review != nil || !strings.Contains(s.Observation, "roomMemberBotCatalogState") {
		t.Fatal("room outcome unlocked", s)
	}
	if _, _, ok = c.Begin("room-account", r); ok {
		t.Fatal("unknown resend")
	}
	c.Bind("member-account", "same-client")
	if c.Snapshot("member-account").Uncertain {
		t.Fatal("wrong target locked")
	}
}
