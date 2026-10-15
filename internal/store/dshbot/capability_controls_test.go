package dshbot

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func capabilityStoreFixture() pluginrpc.DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"","sectionId":"","description":"keep","workspaceId":"/fixture","avatar":{"kind":"blob","shape":"circle","color":"blue"},"model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2}`), &p)
	var policies pluginrpc.DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policies)
	return pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s", Name: "Fixture"}, Revision: 4, Profile: p, Policies: policies, Catalog: pluginrpc.DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []pluginrpc.DshbotCapabilityEntry{{Name: "read"}}, Skills: []pluginrpc.DshbotCapabilityEntry{}, MCP: []pluginrpc.DshbotCapabilityEntry{}}}
}
func capabilityStoreLoaded(t *testing.T) (*CapabilityControls, RoutineScope) {
	t.Helper()
	c := &CapabilityControls{}
	k := RoutineScope{HostKey: "fixture-auth", BotID: "b"}
	c.Bind(k)
	ticket, _ := c.BeginLoad(k)
	if !c.FinishLoad(ticket, capabilityStoreFixture(), true, nil) {
		t.Fatal("load")
	}
	return c, k
}
func TestBotCapabilitiesExplicitReviewAndDraftIsolation(t *testing.T) {
	c, k := capabilityStoreLoaded(t)
	old := c.Snapshot(k).Generation
	c.Change(k, "tools", "mode", "selected")
	c.Change(k, "tools", "toggle", "read")
	if c.ViewCurrent(k, old) {
		t.Fatal("old view valid")
	}
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("saved before impact acknowledgement")
	}
	c.Acknowledge(k, r, true)
	r = c.Snapshot(k).Review
	r.Command.Expected.Profile["description"] = json.RawMessage(`"mutated"`)
	r.Command.Desired["tools"] = pluginrpc.DshbotCapabilityPolicy{Mode: "all", Names: []string{}}
	ticket, cmd, ok := c.BeginSave(k, r)
	if !ok || cmd.Desired["tools"].Mode != "selected" || string(cmd.Expected.Profile["description"]) != `"keep"` {
		t.Fatal("review alias", ok, cmd)
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("double save")
	}
	c.FinishSave(ticket, pluginrpc.DshbotCapabilityReceipt{Attempted: true}, errors.New("synthetic unknown"))
	if c.Snapshot(k).Loaded || !c.Snapshot(k).Dirty {
		t.Fatal("unknown unlocked or lost draft")
	}
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("retry before readback")
	}
	load, _ := c.BeginLoad(k)
	fresh := capabilityStoreFixture()
	fresh.Revision++
	c.FinishLoad(load, fresh, true, nil)
	s := c.Snapshot(k)
	if !s.Dirty || s.Draft["tools"].Mode != "selected" {
		t.Fatal("refresh dropped draft")
	}
	c.Prepare(k)
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("old review consumed new one")
	}
}
func TestBotCapabilitiesReadbackAndScopeFence(t *testing.T) {
	var parent RoutineControls
	k := RoutineScope{HostKey: "a", BotID: "b"}
	parent.Bind(k)
	c := &parent.Capabilities
	ticket, _ := c.BeginLoad(k)
	parent.Bind(RoutineScope{})
	parent.Bind(k)
	if c.FinishLoad(ticket, capabilityStoreFixture(), true, nil) {
		t.Fatal("reopen accepted old load")
	}
	ticket, _ = c.BeginLoad(k)
	c.FinishLoad(ticket, capabilityStoreFixture(), true, nil)
	c.Change(k, "tools", "mode", "selected")
	c.Prepare(k)
	r := c.Snapshot(k).Review
	c.Acknowledge(k, r, true)
	r = c.Snapshot(k).Review
	ticket, cmd, ok := c.BeginSave(k, r)
	if !ok {
		t.Fatal("save")
	}
	receipt := pluginrpc.CloneDshbotCapabilities(cmd.Expected)
	receipt.Policies = pluginrpc.CloneDshbotCapabilityPolicies(cmd.Desired)
	receipt.Profile["capabilities"], _ = json.Marshal(receipt.Policies)
	receipt.Revision++
	c.FinishSave(ticket, pluginrpc.DshbotCapabilityReceipt{Status: receipt, Attempted: true, Warnings: []string{"fixture warning"}}, nil)
	if c.Snapshot(k).Loaded || len(c.Snapshot(k).Warnings) != 1 {
		t.Fatal("receipt skips readback")
	}
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, receipt, true, nil)
	if c.Snapshot(k).Dirty || !c.Snapshot(k).Loaded {
		t.Fatal("readback not reconciled")
	}
	c.Change(k, "tools", "mode", "all")
	c.Prepare(k)
	r = c.Snapshot(k).Review
	c.Acknowledge(k, r, true)
	ticket, _, _ = c.BeginSave(k, c.Snapshot(k).Review)
	parent.Bind(RoutineScope{HostKey: "b", BotID: "b"})
	if c.Current(ticket) {
		t.Fatal("foreign Host dispatch")
	}
}
func TestBotCapabilitiesUnavailableAndReadOnly(t *testing.T) {
	c, k := capabilityStoreLoaded(t)
	load, _ := c.BeginLoad(k)
	bad := capabilityStoreFixture()
	bad.WriteError = "unknown nested model"
	c.FinishLoad(load, bad, true, nil)
	c.Change(k, "tools", "mode", "selected")
	if c.Snapshot(k).Dirty {
		t.Fatal("unsafe profile writable")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, capabilityStoreFixture(), false, nil)
	c.Change(k, "tools", "mode", "selected")
	if c.Snapshot(k).Dirty {
		t.Fatal("read-only authority writable")
	}
}
