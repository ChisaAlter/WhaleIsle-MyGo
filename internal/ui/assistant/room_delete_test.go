package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
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

func TestRoomDeleteUIExplicitRoomConfirmation(t *testing.T) {
	p := roomDeleteFixturePlan(t)
	state := dshbotstore.BotDeleteSnapshot{Scope: "room", Loaded: true, Writable: true, Plan: p, Review: &dshbotstore.BotDeleteReview{Plan: p}}
	writes := 0
	tt := ui.NewTester(func(c *ui.Context) {
		botDeleteView(c, BotDeleteModel{State: state, HostLabel: "room-host", SourceID: "room", OnCancel: func() { state.Review = nil }, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { writes++ }}, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 8000)
	text := strings.Join(tt.Texts(), "\n")
	for _, s := range []string{"kind: room", "name-room", "session-room", "session-a", "session-b", "full room routine prompt", "永久解散", "成员 Bot 目录保留", "不保证可恢复", "非原子", "当前"} {
		if !strings.Contains(text, s) {
			t.Fatal("missing room-specific review", s)
		}
	}
	tt.Click(roomDeleteConfirmLabel)
	if writes != 0 {
		t.Fatal("default room deletion")
	}
	state.Review.TypedID = "a"
	state.Review.TypedName = "name-a"
	state.Review.Acknowledged = true
	tt.Frame()
	tt.Click(roomDeleteConfirmLabel)
	if writes != 0 {
		t.Fatal("member identity accepted")
	}
	state.Review.TypedID = "room"
	state.Review.TypedName = "name-room"
	state.Review.Acknowledged = false
	tt.Frame()
	tt.Click(roomDeleteConfirmLabel)
	if writes != 0 {
		t.Fatal("missing danger consent")
	}
	tt.Click(botDeleteDangerAcknowledgement)
	if err := tt.Click(roomDeleteConfirmLabel); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatal("explicit room confirmation failed", writes)
	}
}
