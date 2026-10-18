package dshbot

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

const roomTextFixtureJSON = `{"id":"room","kind":"room","sessionId":"canonical-room","name":"Fixture group","title":"","description":"preserve full description","workspaceId":"workspace-1","sectionId":"section-1","pinned":false,"hidden":false,"pinOrder":0,"readInitialized":true,"lastSeenSeq":4,"createdAt":1,"updatedAt":2,"objective":"","objectiveMaxRounds":16,"memoryReview":true,"notify":true,"allowedSenderIds":[],"memberBotIds":["a","b"],"maxRounds":3,"maxSpeaks":10,"avatar":{"kind":"blob","shape":"circle","color":"blue","crop":"","dataUrl":""},"model":{"provider":"","model":"","reasoningEffort":""},"capabilities":{"tools":{"mode":"all","names":[],"future":{"guard":true}},"skills":{"mode":"all","names":[]},"mcp":{"mode":"all","names":[]},"extra":{"nested":[1,2]}},"holds":{},"inbox":[],"futureTop":{"retain":{"bytes":"value"}}}`

func roomTextFixtureStatus() pluginrpc.DshbotRoomEditStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(roomTextFixtureJSON), &p)
	var row pluginrpc.DshbotRoomItem
	json.Unmarshal([]byte(roomTextFixtureJSON), &row)
	return pluginrpc.DshbotRoomEditStatus{Profile: p, Item: row, Sections: []string{"section-1"}, Catalog: pluginrpc.DshbotRoomCatalog{Revision: 7, Items: []pluginrpc.DshbotRoomItem{row, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: "bot", SessionID: "sa"}}, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "sb"}}}}}
}

func TestRoomTextDraftLifecycleAndUnknownWarnings(t *testing.T) {
	var c RoomEditor
	s := roomTextFixtureStatus()
	c.Bind("room-account", "client")
	load, _ := c.BeginLoad("room-account")
	c.FinishLoad(load, s, true, nil)
	c.Start("room-account")
	if c.Snapshot("room-account").Draft.Name != s.Item.Name {
		t.Fatal("missing original text")
	}
	c.Prepare("room-account")
	if c.Snapshot("room-account").Review != nil {
		t.Fatal("unchanged text save")
	}
	c.Change("room-account", "name", " New group ")
	c.Change("room-account", "description", " First line\nSecond line ")
	c.Prepare("room-account")
	r := c.Snapshot("room-account").Review
	if r == nil || r.Command.Values.Text.Name != "New group" || r.Command.Values.Text.Description != "First line\nSecond line" {
		t.Fatal("normalized review", c.Snapshot("room-account"))
	}
	if _, _, ok := c.Begin("room-account", r); ok {
		t.Fatal("unconfirmed presentation effect")
	}
	c.Acknowledge("room-account", r, true)
	ticket, cmd, ok := c.Begin("room-account", r)
	if !ok || !c.MarkSent(ticket) || c.MarkSent(ticket) {
		t.Fatal("single dispatch")
	}
	c.Suspend()
	c.Bind("other", "other-client")
	c.Finish(ticket, pluginrpc.DshbotRoomEditResult{Attempted: true, Warnings: []string{"rename may have failed completely"}}, fmt.Errorf("unknown receipt"))
	if c.Snapshot("other").Uncertain {
		t.Fatal("wrong account settled")
	}
	c.Bind("room-account", "new-client")
	c.Discard("room-account")
	if _, _, ok := c.Begin("room-account", r); ok {
		t.Fatal("unknown resent")
	}
	pending := c.Snapshot("room-account")
	if !pending.Uncertain || pending.Draft == nil || len(pending.Warnings) != 1 {
		t.Fatal("lost unknown draft/warning", pending)
	}
	fresh := pluginrpc.CloneDshbotRoomEdit(s)
	fresh.Catalog.Revision++
	fresh.Profile = pluginrpc.DshbotRoomEditProfile(cmd)
	raw, _ := json.Marshal(fresh.Profile)
	json.Unmarshal(raw, &fresh.Item)
	load, _ = c.BeginLoad("room-account")
	c.FinishLoad(load, fresh, true, nil)
	if len(c.Snapshot("room-account").Warnings) != 1 || !c.Snapshot("room-account").Candidate {
		t.Fatal("readback lost warning")
	}
	if !c.Adopt("room-account") || !strings.Contains(c.Snapshot("room-account").Notice, "副作用") {
		t.Fatal("readback falsely proved rename")
	}
	c.Start("room-account")
	c.Change("room-account", "description", "newer")
	c.Prepare("room-account")
	old := c.Snapshot("room-account").Review
	c.Suspend()
	c.Bind("room-account", "new-client")
	if _, _, ok := c.Begin("room-account", old); ok {
		t.Fatal("hidden stale review survived")
	}
}
