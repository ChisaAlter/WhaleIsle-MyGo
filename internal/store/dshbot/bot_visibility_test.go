package dshbot

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func visibilityCatalog() pluginrpc.DshbotBotCatalog {
	v := pluginrpc.DshbotBotCatalog{Revision: 8}
	for _, x := range []struct {
		id, kind, sid string
		hidden        bool
	}{{"b", "bot", "s", false}, {"r", "room", "rs", true}} {
		raw, _ := json.Marshal(map[string]any{"id": x.id, "kind": x.kind, "sessionId": x.sid, "name": "Fixture", "hidden": x.hidden, "updatedAt": 2, "unknown": map[string]any{"keep": true}})
		var item pluginrpc.DshbotBotItem
		json.Unmarshal(raw, &item.DshbotCatalogItem)
		json.Unmarshal(raw, &item.Profile)
		v.Items = append(v.Items, item)
	}
	return v
}
func visibilityRead(t *testing.T, c *BotVisibilityManager, k string, v pluginrpc.DshbotBotCatalog, w bool) {
	t.Helper()
	ticket, ok := c.BeginLoad(k)
	if !ok || !c.FinishLoad(ticket, v, w, nil) {
		t.Fatal("read")
	}
}
func visibilityCommand(t *testing.T, c *BotVisibilityManager, k, id, sid string) (BotVisibilityTicket, pluginrpc.DshbotVisibilityCommand) {
	t.Helper()
	c.Prepare(k, id, sid, 100)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal("review")
	}
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("missing ack")
	}
	c.Acknowledge(k, r, true)
	ticket, cmd, ok := c.Begin(k, c.Snapshot(k).Review)
	if !ok {
		t.Fatal("begin")
	}
	return ticket, cmd
}
func visibilityResult(t *testing.T, cmd pluginrpc.DshbotVisibilityCommand) pluginrpc.DshbotVisibilityResult {
	t.Helper()
	v := pluginrpc.CloneDshbotBotCatalog(cmd.Expected)
	v.Revision++
	for i := range v.Items {
		if v.Items[i].ID == cmd.ID {
			v.Items[i].Hidden = cmd.Hidden
			v.Items[i].Profile["hidden"], _ = json.Marshal(cmd.Hidden)
			v.Items[i].Profile["updatedAt"], _ = json.Marshal(cmd.UpdatedAt)
		}
	}
	row, err := pluginrpc.DshbotVisibilityApplied(cmd, v)
	if err != nil {
		t.Fatal(err)
	}
	return pluginrpc.DshbotVisibilityResult{Attempted: true, Catalog: v, Item: row}
}
func TestBotVisibilityHideRevealRestoreAndUnknown(t *testing.T) {
	var c BotVisibilityManager
	c.Bind("A", "client1")
	base := visibilityCatalog()
	visibilityRead(t, &c, "A", base, true)
	if !c.Reveal("A", "r", "rs") || !c.Snapshot("A").ShowHidden || c.Snapshot("A").FocusID != "r" {
		t.Fatal("local exact reveal")
	}
	if !pluginrpc.SameDshbotBotCatalog(base, c.Snapshot("A").Catalog) {
		t.Fatal("reveal modified Host state")
	}
	ticket, cmd := visibilityCommand(t, &c, "A", "b", "s")
	if !c.MarkSent(ticket) || c.MarkSent(ticket) {
		t.Fatal("once")
	}
	result := visibilityResult(t, cmd)
	c.Finish(ticket, result, nil)
	s := c.Snapshot("A")
	if s.Result.ID != "b" || !s.Result.Hidden || !s.ShowHidden || s.Uncertain {
		t.Fatal("hidden outcome not locatable")
	}
	c.Show("A", false)
	if c.Snapshot("A").Result.ID != "b" {
		t.Fatal("filter dropped result")
	}
	visibilityRead(t, &c, "A", result.Catalog, true)
	ticket, cmd = visibilityCommand(t, &c, "A", "b", "s")
	if cmd.Hidden {
		t.Fatal("restore draft")
	}
	c.MarkSent(ticket)
	c.Finish(ticket, visibilityResult(t, cmd), nil)
	if c.Snapshot("A").Result.Hidden {
		t.Fatal("restore")
	}
	visibilityRead(t, &c, "A", base, true)
	ticket, cmd = visibilityCommand(t, &c, "A", "r", "rs")
	c.MarkSent(ticket)
	c.Suspend()
	c.Bind("B", "client2")
	c.Finish(ticket, pluginrpc.DshbotVisibilityResult{Attempted: true}, errors.New("reply lost"))
	if c.Snapshot("B").Uncertain || !c.Snapshot("A").Uncertain {
		t.Fatal("late settle crossed account")
	}
	c.Bind("A", "new-client")
	s = c.Snapshot("A")
	if s.Pending == nil || s.Pending.ID != "r" || s.Pending.SessionID != "rs" {
		t.Fatal("pending identity lost")
	}
	c.Prepare("A", "b", "s", 101)
	if c.Snapshot("A").Review != nil {
		t.Fatal("unknown allowed write")
	}
	visibilityRead(t, &c, "A", base, true)
	if c.Snapshot("A").Candidate != nil || c.Adopt("A", "r", "rs") {
		t.Fatal("old state unlocked attempt")
	}
	result = visibilityResult(t, cmd)
	visibilityRead(t, &c, "A", result.Catalog, false)
	if c.Snapshot("A").Candidate == nil || !c.Snapshot("A").Uncertain {
		t.Fatal("readback must not auto adopt")
	}
	if c.Adopt("A", "r", "wrong") || !c.Adopt("A", "r", "rs") {
		t.Fatal("exact explicit adopt")
	}
	if c.Snapshot("A").Uncertain || c.Snapshot("A").Result.ID != "r" {
		t.Fatal("adoption")
	}
	visibilityRead(t, &c, "A", base, false)
	c.Prepare("A", "b", "s", 100)
	if c.Snapshot("A").Review != nil {
		t.Fatal("readonly")
	}
	visibilityRead(t, &c, "A", base, true)
	ticket, _ = visibilityCommand(t, &c, "A", "b", "s")
	c.Suspend()
	if c.MarkSent(ticket) {
		t.Fatal("hidden preflight sent")
	}
	c.AbortUnsent(ticket, nil)
	c.Bind("A", "client1")
	if c.Snapshot("A").Busy || c.Snapshot("A").Pending != nil {
		t.Fatal("unsent stuck")
	}
	visibilityRead(t, &c, "A", base, true)
	c.Prepare("A", "b", "s", 100)
	stale := c.Snapshot("A").Review
	visibilityRead(t, &c, "A", base, true)
	c.Acknowledge("A", stale, true)
	if _, _, ok := c.Begin("A", stale); ok {
		t.Fatal("stale review")
	}
	copy := c.Snapshot("A")
	copy.Catalog.Items[0].Profile["hidden"] = json.RawMessage(`true`)
	if string(c.Snapshot("A").Catalog.Items[0].Profile["hidden"]) != "false" {
		t.Fatal("snapshot alias")
	}
}
