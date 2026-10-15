package dshbot

import (
	"encoding/json"
	"fmt"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func duplicateStoreCatalog() pluginrpc.DshbotBotCatalog {
	var item pluginrpc.DshbotBotItem
	raw := []byte(`{"id":"source","kind":"bot","sessionId":"source-session","name":"Source","title":"Display title","sectionId":"","description":"Full preserved description","avatar":{"kind":"blob","shape":"circle","color":"blue"},"workspaceId":"workspace-fixture","model":{"provider":"provider","model":"model","reasoningEffort":"high"},"pinned":true,"hidden":false,"pinOrder":7,"allowedSenderIds":[],"memberBotIds":[],"objective":"Complete this entire objective","objectiveMaxRounds":23,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"selected","names":["read"]},"skills":{"mode":"all","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":2,"maxSpeaks":7,"inbox":[{"id":"old-message"}],"readInitialized":true,"lastSeenSeq":8,"createdAt":1,"updatedAt":2}`)
	json.Unmarshal(raw, &item.DshbotCatalogItem)
	json.Unmarshal(raw, &item.Profile)
	return pluginrpc.DshbotBotCatalog{Revision: 7, Items: []pluginrpc.DshbotBotItem{item}}
}
func duplicateStoreReady(t *testing.T) (*BotDuplicator, string) {
	c := &BotDuplicator{}
	k := "account-a/source"
	c.Bind(k, "client-a")
	load, ok := c.BeginLoad(k)
	if !ok {
		t.Fatal("load")
	}
	c.FinishLoad(load, duplicateStoreCatalog(), true, nil)
	c.Start(k, "source", "/reviewed/cwd")
	c.Change(k, "name", "Chosen copy name")
	return c, k
}
func duplicateStoreBegin(t *testing.T, c *BotDuplicator, k string) (BotDuplicateTicket, pluginrpc.DshbotBotDuplicateCommand) {
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	c.Acknowledge(k, r, true)
	r = c.Snapshot(k).Review
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("goal copied without separate execution acknowledgement")
	}
	c.AcknowledgeExecution(k, r, true)
	r = c.Snapshot(k).Review
	ticket, cmd, ok := c.Begin(k, r)
	if !ok || !cmd.AllowObjectiveExecution {
		t.Fatal("explicit begin")
	}
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("duplicate submit")
	}
	if !c.MarkSent(ticket) || c.MarkSent(ticket) {
		t.Fatal("duplicate dispatch")
	}
	return ticket, cmd
}
func duplicateStoreResult(cmd pluginrpc.DshbotBotDuplicateCommand) pluginrpc.DshbotBotDuplicateResult {
	p := map[string]json.RawMessage{}
	for _, key := range pluginrpc.DshbotBotDuplicateConfigKeys {
		p[key] = append(json.RawMessage(nil), cmd.Expected.Items[0].Profile[key]...)
	}
	for key, v := range map[string]any{"id": "new-copy", "sessionId": "new-session", "kind": "bot", "name": cmd.Name, "hidden": false, "pinned": false, "pinOrder": 0, "inbox": []any{}, "memberBotIds": []string{}, "readInitialized": true, "lastSeenSeq": 0, "createdAt": 3, "updatedAt": 3, "maxRounds": 3, "maxSpeaks": 10} {
		p[key], _ = json.Marshal(v)
	}
	raw, _ := json.Marshal(p)
	var row pluginrpc.DshbotBotItem
	json.Unmarshal(raw, &row.DshbotCatalogItem)
	row.Profile = p
	view := pluginrpc.CloneDshbotBotCatalog(cmd.Expected)
	view.Revision++
	view.Items = append(view.Items, row)
	return pluginrpc.DshbotBotDuplicateResult{Attempted: true, Item: row, Catalog: view, Warnings: []string{"Memory copy failed: keep this entire warning"}}
}
func TestBotDuplicatorUnknownNeverAdoptsCandidate(t *testing.T) {
	c, k := duplicateStoreReady(t)
	ticket, cmd := duplicateStoreBegin(t, c, k)
	c.Suspend()
	c.Bind("other-account/source", "client-b")
	c.Finish(ticket, pluginrpc.DshbotBotDuplicateResult{Attempted: true}, fmt.Errorf("lost receipt"))
	if c.Snapshot("other-account/source").Draft != nil {
		t.Fatal("cross-account leak")
	}
	c.Bind(k, "client-c")
	c.Discard(k)
	c.Start(k, "source", "/other")
	if !c.Snapshot(k).Uncertain || c.Snapshot(k).Draft.Name != cmd.Name {
		t.Fatal("unknown unlocked")
	}
	load, _ := c.BeginLoad(k)
	empty := duplicateStoreCatalog()
	empty.Revision++
	c.FinishLoad(load, empty, true, nil)
	c.Prepare(k)
	if c.Snapshot(k).Review != nil || len(c.Snapshot(k).Candidates) != 0 {
		t.Fatal("absence authorized another copy")
	}
	result := duplicateStoreResult(cmd)
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, result.Catalog, true, nil)
	snapshot := c.Snapshot(k)
	if !snapshot.Uncertain || len(snapshot.Candidates) != 1 || snapshot.Created.ID != "" || snapshot.Pending == nil {
		t.Fatal("candidate causally adopted", snapshot)
	}
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("candidate unlocked duplicate")
	}
	c.Suspend()
	c.Bind(k, "client-c")
	if c.ViewCurrent(k, snapshot.Generation) {
		t.Fatal("hidden/reopened revived stale action")
	}
}
func TestBotDuplicatorReviewLifecycleAndWarnings(t *testing.T) {
	c, k := duplicateStoreReady(t)
	c.Prepare(k)
	r := c.Snapshot(k).Review
	c.Acknowledge(k, r, true)
	c.AcknowledgeExecution(k, r, true)
	c.Change(k, "name", "Changed name")
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("stale review")
	}
	ticket, cmd := duplicateStoreBegin(t, c, k)
	c.Suspend()
	c.Finish(ticket, duplicateStoreResult(cmd), nil)
	c.Bind(k, "replacement")
	s := c.Snapshot(k)
	if s.Uncertain || s.Busy || s.Created.ID != "new-copy" || s.CreatedScratchCwd != "/reviewed/cwd" || len(s.Warnings) != 1 || s.Warnings[0] != "Memory copy failed: keep this entire warning" {
		t.Fatal("lost independent result/warning", s)
	}
	load, _ := c.BeginLoad(k)
	c.Suspend()
	c.Bind(k, "replacement")
	if c.FinishLoad(load, duplicateStoreCatalog(), true, nil) {
		t.Fatal("late load")
	}
}
