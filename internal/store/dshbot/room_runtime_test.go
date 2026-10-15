package dshbot

import (
	"encoding/base64"
	"fmt"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestRoomRuntimeReaderViewAndFailureFixture(t *testing.T) {
	c := &RoomRuntimeReader{}
	c.Bind("original-client-room")
	s := c.Snapshot()
	ticket, ok := c.BeginRead(s.Scope, s.Generation)
	if !ok {
		t.Fatal("read blocked")
	}
	sid := "dshbot-group-member-v3-" + base64.RawURLEncoding.EncodeToString([]byte(`["room-session","a"]`))
	data := pluginrpc.DshbotRoomRuntime{Room: pluginrpc.DshbotRoomItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "room", Kind: "room", SessionID: "room-session", MemberBotIDs: []string{"a", "b"}}}, Catalog: pluginrpc.DshbotRoomCatalog{Items: []pluginrpc.DshbotRoomItem{{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: "bot", SessionID: "personal"}}}}, Members: []pluginrpc.DshbotRoomRuntimeMember{{RoomSessionID: "room-session", BotID: "a", SessionID: sid, Status: "pending"}}}
	c.FinishRead(ticket, data, nil)
	if !c.Snapshot().Loaded {
		t.Fatal("read not applied")
	}
	open, _, ok := c.BeginOpen(s.Scope, s.Generation, sid)
	if !ok || !c.Current(open) {
		t.Fatal("explicit member open unavailable")
	}
	if _, _, ok = c.BeginOpen(s.Scope, s.Generation, sid); ok {
		t.Fatal("double open admitted")
	}
	c.Suspend()
	c.Bind(s.Scope)
	if c.Current(open) || c.ViewCurrent(s.Scope, s.Generation) || c.FinishOpen(open, nil) {
		t.Fatal("old view revived")
	}
	s = c.Snapshot()
	ticket, _ = c.BeginRead(s.Scope, s.Generation)
	c.FinishRead(ticket, data, nil)
	ticket, _ = c.BeginRead(s.Scope, s.Generation)
	c.FinishRead(ticket, pluginrpc.DshbotRoomRuntime{}, fmt.Errorf("group/runtime failed"))
	failed := c.Snapshot()
	if failed.Loaded || failed.Error == "" || failed.Notice == "" || failed.Data.Room.ID != "room" {
		t.Fatal("failure hidden or treated as healthy", failed)
	}
	if _, _, ok = c.BeginOpen(s.Scope, s.Generation, sid); ok {
		t.Fatal("stale failed snapshot opened")
	}
	c.Bind("replacement-client-room")
	if c.Snapshot().Data.Room.ID != "" {
		t.Fatal("cross-account runtime leaked")
	}
}
