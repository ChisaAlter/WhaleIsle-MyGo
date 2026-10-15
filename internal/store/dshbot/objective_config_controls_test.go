package dshbot

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func objectiveConfigStoreFixture() pluginrpc.DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"","sectionId":"","description":"Keep","workspaceId":"/fixture","avatar":{"kind":"blob","shape":"circle","color":"blue"},"model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2}`), &p)
	var policy pluginrpc.DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	return pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 2, Profile: p, Policies: policy, Catalog: pluginrpc.DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []pluginrpc.DshbotCapabilityEntry{}, Skills: []pluginrpc.DshbotCapabilityEntry{}, MCP: []pluginrpc.DshbotCapabilityEntry{}}}
}
func objectiveConfigLoaded(t *testing.T) (*ObjectiveConfigControls, RoutineScope) {
	t.Helper()
	c := &ObjectiveConfigControls{}
	k := RoutineScope{HostKey: "fixture-auth", BotID: "b"}
	c.Bind(k)
	load, _ := c.BeginLoad(k)
	if !c.FinishLoad(load, objectiveConfigStoreFixture(), true, nil) {
		t.Fatal("load")
	}
	return c, k
}
func TestBotObjectiveConfigIndependentConfirmationAndUnknownDraft(t *testing.T) {
	c, k := objectiveConfigLoaded(t)
	c.SetDraft(k, pluginrpc.DshbotObjectiveConfigDraft{Objective: "  Changed objective  ", MaxRounds: "20"})
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("unacknowledged save")
	}
	c.Acknowledge(k, r, true)
	r = c.Snapshot(k).Review
	ticket, cmd, ok := c.BeginSave(k, r)
	if !ok || cmd.Desired.Objective != "Changed objective" || cmd.Desired.MaxRounds != 20 {
		t.Fatal("review lost")
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("double send")
	}
	c.FinishSave(ticket, pluginrpc.DshbotCapabilityReceipt{Attempted: true}, errors.New("synthetic uncertain"))
	s := c.Snapshot(k)
	if s.Loaded || s.Draft.Objective != "  Changed objective  " || !s.Dirty {
		t.Fatal("unknown lost/unlocked draft")
	}
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("save before readback")
	}
	load, _ := c.BeginLoad(k)
	fresh := objectiveConfigStoreFixture()
	fresh.Revision++
	c.FinishLoad(load, fresh, true, nil)
	c.Prepare(k)
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("old review reused")
	}
}
func TestBotObjectiveConfigScopeAndReadback(t *testing.T) {
	var parent RoutineControls
	k := RoutineScope{HostKey: "a", BotID: "b"}
	parent.Bind(k)
	c := &parent.ObjectiveConfig
	ticket, _ := c.BeginLoad(k)
	oldGeneration := c.Snapshot(k).Generation
	parent.Bind(RoutineScope{})
	parent.Bind(k)
	if c.ViewCurrent(k, oldGeneration) || c.FinishLoad(ticket, objectiveConfigStoreFixture(), true, nil) {
		t.Fatal("old load survived reopen")
	}
	ticket, _ = c.BeginLoad(k)
	c.FinishLoad(ticket, objectiveConfigStoreFixture(), true, nil)
	c.SetDraft(k, pluginrpc.DshbotObjectiveConfigDraft{Objective: "  Changed objective  ", MaxRounds: "20"})
	c.Prepare(k)
	c.Acknowledge(k, c.Snapshot(k).Review, true)
	ticket, _, ok := c.BeginSave(k, c.Snapshot(k).Review)
	if !ok {
		t.Fatal("save missing")
	}
	out := objectiveConfigStoreFixture()
	out.Profile["objective"] = json.RawMessage(`"Changed objective"`)
	out.Profile["objectiveMaxRounds"] = json.RawMessage(`20`)
	out.Revision++
	c.FinishSave(ticket, pluginrpc.DshbotCapabilityReceipt{Status: out, Attempted: true, Warnings: []string{"fixture warning"}}, nil)
	if c.Snapshot(k).Loaded {
		t.Fatal("receipt skipped readback")
	}
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, out, true, nil)
	if c.Snapshot(k).Dirty || c.Snapshot(k).Draft.Objective != "Changed objective" || len(c.Snapshot(k).Warnings) != 1 {
		t.Fatal("readback not reconciled")
	}
	parent.Bind(RoutineScope{HostKey: "b", BotID: "b"})
	if c.Current(ticket) {
		t.Fatal("foreign Host ticket")
	}
}
func TestBotObjectiveConfigMissingCapAndReadOnly(t *testing.T) {
	c, k := objectiveConfigLoaded(t)
	load, _ := c.BeginLoad(k)
	value := objectiveConfigStoreFixture()
	delete(value.Profile, "objectiveMaxRounds")
	c.FinishLoad(load, value, true, nil)
	if c.Snapshot(k).Loaded {
		t.Fatal("missing cap defaulted")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, objectiveConfigStoreFixture(), false, nil)
	c.SetDraft(k, pluginrpc.DshbotObjectiveConfigDraft{Objective: "  Changed objective  ", MaxRounds: "20"})
	if c.Snapshot(k).Dirty {
		t.Fatal("readonly draft changed")
	}
}
