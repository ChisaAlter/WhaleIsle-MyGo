package dshbot

import (
	"encoding/json"
	"fmt"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func botCreatorCatalog() pluginrpc.DshbotBotCatalog {
	return pluginrpc.DshbotBotCatalog{Revision: 3, Items: []pluginrpc.DshbotBotItem{}}
}
func botCreatorReady(t *testing.T) (*BotCreator, string) {
	c := &BotCreator{}
	k := "account-a"
	c.Bind(k, "client-a")
	load, ok := c.BeginLoad(k)
	if !ok {
		t.Fatal("load")
	}
	c.FinishLoad(load, botCreatorCatalog(), true, nil)
	c.Start(k, "/reviewed-cwd")
	c.Change(k, "provider", "provider")
	c.Change(k, "model", "model")
	return c, k
}
func botCreatorBegin(t *testing.T, c *BotCreator, k string) (BotCreateTicket, pluginrpc.DshbotBotCreateCommand) {
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("no acknowledgement")
	}
	c.Acknowledge(k, r, true)
	r = c.Snapshot(k).Review
	ticket, cmd, ok := c.Begin(k, r)
	if !ok {
		t.Fatal("begin")
	}
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("double begin")
	}
	if !c.MarkSent(ticket) || c.MarkSent(ticket) {
		t.Fatal("double send")
	}
	return ticket, cmd
}
func botCreatorCreated(cmd pluginrpc.DshbotBotCreateCommand) pluginrpc.DshbotBotCreateResult {
	d := cmd.Draft
	p := map[string]any{"id": cmd.ID, "sessionId": cmd.SessionID, "kind": "bot", "name": d.Name, "title": d.Title, "description": d.Description, "workspaceId": d.WorkspaceID, "sectionId": "", "model": map[string]string{"provider": d.Provider, "model": d.Model, "reasoningEffort": d.ReasoningEffort}, "avatar": map[string]string{"kind": "blob", "shape": "circle", "color": "blue"}, "capabilities": map[string]any{"tools": map[string]any{"mode": "all", "names": []string{}}, "skills": map[string]any{"mode": "all", "names": []string{}}, "mcp": map[string]any{"mode": "all", "names": []string{}}}, "allowedSenderIds": []string{}, "memberBotIds": []string{}, "objective": "", "objectiveMaxRounds": 16, "memoryReview": d.MemoryReview, "notify": d.Notify, "maxRounds": 3, "maxSpeaks": 10, "hidden": false, "pinned": false}
	raw, _ := json.Marshal(p)
	var profile map[string]json.RawMessage
	json.Unmarshal(raw, &profile)
	row := pluginrpc.DshbotBotItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: cmd.ID, SessionID: cmd.SessionID, Kind: "bot", Name: d.Name, Title: d.Title, WorkspaceID: d.WorkspaceID}, Profile: profile}
	return pluginrpc.DshbotBotCreateResult{Attempted: true, Item: row, Catalog: pluginrpc.DshbotBotCatalog{Revision: cmd.Expected.Revision + 1, Items: []pluginrpc.DshbotBotItem{row}}}
}
func TestBotCreatorLifecycleUnknown(t *testing.T) {
	for _, event := range []string{"same-frame-leave", "hide", "minimize", "close", "host-account-generation"} {
		t.Run(event, func(t *testing.T) {
			c, k := botCreatorReady(t)
			c.Prepare(k)
			old := c.Snapshot(k)
			c.Acknowledge(k, old.Review, true)
			c.Suspend()
			c.Bind(k, "client-a")
			if c.ViewCurrent(k, old.Generation) {
				t.Fatal("revived callback")
			}
			if _, _, ok := c.Begin(k, old.Review); ok {
				t.Fatal("stale review")
			}
			load, _ := c.BeginLoad(k)
			c.FinishLoad(load, botCreatorCatalog(), true, nil)
			ticket, cmd := botCreatorBegin(t, c, k)
			c.Suspend()
			c.Bind("account-b", "client-b")
			c.Finish(ticket, pluginrpc.DshbotBotCreateResult{Attempted: true}, fmt.Errorf("lost"))
			if c.Snapshot("account-b").Draft != nil {
				t.Fatal("cross-account draft")
			}
			c.Bind(k, "client-c")
			c.Discard(k)
			c.Start(k, "/different")
			if s := c.Snapshot(k); !s.Uncertain || s.Busy || s.Draft.ID != cmd.ID {
				t.Fatal("lost attempt", s)
			}
			load, _ = c.BeginLoad(k)
			empty := botCreatorCatalog()
			empty.Revision += 2
			c.FinishLoad(load, empty, true, nil)
			c.Prepare(k)
			if c.Snapshot(k).Review != nil || c.Snapshot(k).Candidate != nil {
				t.Fatal("absence released unknown")
			}
			result := botCreatorCreated(cmd)
			load, _ = c.BeginLoad(k)
			c.FinishLoad(load, result.Catalog, true, nil)
			if c.Snapshot(k).Candidate == nil || !c.Adopt(k, cmd.ID) {
				t.Fatal("exact adopt")
			}
			s := c.Snapshot(k)
			if s.Created.ID != cmd.ID || s.CreatedScratchCwd != "/reviewed-cwd" || s.Draft != nil {
				t.Fatal("adopt lost reviewed identity/cwd", s)
			}
		})
	}
}
func TestBotCreatorReviewPreflightAndSuccess(t *testing.T) {
	c, k := botCreatorReady(t)
	c.Prepare(k)
	r := c.Snapshot(k).Review
	c.Acknowledge(k, r, true)
	c.Change(k, "title", "new title")
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("stale form review")
	}
	ticket, cmd := botCreatorBegin(t, c, k)
	c.Finish(ticket, pluginrpc.DshbotBotCreateResult{Attempted: true, Rejected: true}, fmt.Errorf("read-only"))
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("rejection must refresh")
	}
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, botCreatorCatalog(), false, nil)
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("read-only review")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, botCreatorCatalog(), true, nil)
	ticket, cmd = botCreatorBegin(t, c, k)
	c.Suspend()
	c.Finish(ticket, botCreatorCreated(cmd), nil)
	c.Bind(k, "client-new")
	if s := c.Snapshot(k); s.Created.ID != cmd.ID || s.CreatedScratchCwd != "/reviewed-cwd" || s.Uncertain || s.Busy {
		t.Fatal("late success failed to settle original", s)
	}
	load, _ = c.BeginLoad(k)
	c.Suspend()
	c.Bind(k, "client-new")
	if c.FinishLoad(load, botCreatorCatalog(), true, nil) {
		t.Fatal("stale load")
	}
}
