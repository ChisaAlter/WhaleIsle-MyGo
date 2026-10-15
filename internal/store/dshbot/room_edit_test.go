package dshbot

import (
	"encoding/json"
	"fmt"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

const roomEditStoreJSON = `{"id":"room","kind":"room","sessionId":"canonical-room","name":"Fixture group","title":"","description":"preserve full description","workspaceId":"workspace-1","sectionId":"section-1","pinned":false,"hidden":false,"pinOrder":0,"readInitialized":true,"lastSeenSeq":4,"createdAt":1,"updatedAt":2,"objective":"","objectiveMaxRounds":16,"memoryReview":true,"notify":true,"allowedSenderIds":[],"memberBotIds":["a","b"],"maxRounds":3,"maxSpeaks":10,"avatar":{"kind":"blob","shape":"circle","color":"blue","crop":"","dataUrl":""},"model":{"provider":"","model":"","reasoningEffort":""},"capabilities":{"tools":{"mode":"all","names":[],"future":{"guard":true}},"skills":{"mode":"all","names":[]},"mcp":{"mode":"all","names":[]},"extra":{"nested":[1,2]}},"holds":{},"inbox":[],"futureTop":{"retain":{"bytes":"value"}}}`

func roomEditStoreStatus() pluginrpc.DshbotRoomEditStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(roomEditStoreJSON), &p)
	var row pluginrpc.DshbotRoomItem
	json.Unmarshal([]byte(roomEditStoreJSON), &row)
	return pluginrpc.DshbotRoomEditStatus{Profile: p, Item: row, Sections: []string{"section-1"}, Catalog: pluginrpc.DshbotRoomCatalog{Revision: 7, Items: []pluginrpc.DshbotRoomItem{row, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: "bot", SessionID: "sa"}}, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "sb"}}}}}
}
func TestRoomEditorDraftLifecycleAndUnknownReadback(t *testing.T) {
	c := &RoomEditor{}
	k := "account-room"
	c.Bind(k, "client-1")
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, roomEditStoreStatus(), true, nil)
	c.Start(k)
	c.Change(k, "maxRounds", "2")
	c.Prepare(k)
	review := c.Snapshot(k).Review
	if review == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	if _, _, ok := c.Begin(k, review); ok {
		t.Fatal("missing explicit acknowledgment")
	}
	c.Acknowledge(k, review, true)
	old := c.Snapshot(k)
	c.Suspend()
	c.Bind(k, "client-1")
	if c.ViewCurrent(k, old.Generation) {
		t.Fatal("retired view revived")
	}
	if _, _, ok := c.Begin(k, old.Review); ok {
		t.Fatal("old confirmation revived")
	}
	if c.Snapshot(k).Draft.MaxRounds != "2" {
		t.Fatal("draft lost")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, roomEditStoreStatus(), true, nil)
	c.Prepare(k)
	review = c.Snapshot(k).Review
	c.Acknowledge(k, review, true)
	review = c.Snapshot(k).Review
	ticket, cmd, ok := c.Begin(k, review)
	if !ok || !c.MarkSent(ticket) {
		t.Fatal("explicit save unavailable")
	}
	if _, _, ok = c.Begin(k, review); ok {
		t.Fatal("duplicate send")
	}
	c.Bind("other-account-room", "client-2")
	c.Finish(ticket, pluginrpc.DshbotRoomEditResult{Attempted: true}, fmt.Errorf("lost response"))
	if c.Snapshot("other-account-room").Draft != nil {
		t.Fatal("cross-account result")
	}
	c.Bind(k, "client-3")
	if !c.Snapshot(k).Uncertain {
		t.Fatal("unknown attempt lost")
	}
	c.Discard(k)
	c.Change(k, "maxRounds", "1")
	if c.Snapshot(k).Draft.MaxRounds != "2" {
		t.Fatal("unknown draft unlocked")
	}
	load, _ = c.BeginLoad(k)
	c.Suspend()
	c.Bind(k, "client-3")
	if c.Snapshot(k).Busy {
		t.Fatal("retired unknown readback wedged busy")
	}
	if c.FinishLoad(load, roomEditStoreStatus(), true, nil) {
		t.Fatal("stale read installed")
	}
	load, _ = c.BeginLoad(k)
	fresh := roomEditStoreStatus()
	fresh.Catalog.Revision++
	c.FinishLoad(load, fresh, true, nil)
	c.Prepare(k)
	if c.Snapshot(k).Candidate || c.Snapshot(k).Review != nil || c.Adopt(k) {
		t.Fatal("different readback unlocked")
	}
	fresh.Profile = pluginrpc.DshbotRoomEditProfile(cmd)
	raw, _ := json.Marshal(fresh.Profile)
	json.Unmarshal(raw, &fresh.Item)
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, fresh, true, nil)
	if !c.Snapshot(k).Candidate || !c.Adopt(k) || c.Snapshot(k).Uncertain {
		t.Fatal("exact readback not adopted")
	}
	c.Start(k)
	c.Change(k, "maxRounds", "1")
	c.Prepare(k)
	review = c.Snapshot(k).Review
	c.Acknowledge(k, review, true)
	review = c.Snapshot(k).Review
	ticket, _, ok = c.Begin(k, review)
	if !ok {
		t.Fatal("new reviewed edit unavailable")
	}
	c.Suspend()
	c.AbortUnsent(ticket, fmt.Errorf("view retired"))
	c.Bind(k, "client-4")
	if c.Snapshot(k).Uncertain || c.Snapshot(k).Draft == nil || c.Snapshot(k).Busy {
		t.Fatal("unsent draft not retained")
	}
}
