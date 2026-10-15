package dshbot

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func textProfileStoreFixture() pluginrpc.DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"","sectionId":"","description":"Keep","workspaceId":"/fixture","avatar":{"kind":"blob","shape":"circle","color":"blue"},"model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2}`), &p)
	var policy pluginrpc.DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	return pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 2, Profile: p, Policies: policy, Catalog: pluginrpc.DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []pluginrpc.DshbotCapabilityEntry{}, Skills: []pluginrpc.DshbotCapabilityEntry{}, MCP: []pluginrpc.DshbotCapabilityEntry{}}}
}

func TestBotTextProfileNormalizedReviewUnknownAndScope(t *testing.T) {
	var parent RoutineControls
	k := RoutineScope{HostKey: "fixture-account", BotID: "b"}
	parent.Bind(k)
	c := &parent.TextProfile
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, textProfileStoreFixture(), true, nil)
	v := pluginrpc.DshbotTextProfileValues{Name: "  address-new  ", Title: "  Display new  ", Description: "  Persona new  "}
	c.SetDraft(k, v)
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil || r.Command.Values.Name != "address-new" || r.Command.Values.Title != "Display new" || r.Command.Values.Description != "Persona new" {
		t.Fatal(c.Snapshot(k))
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("unacknowledged")
	}
	c.Acknowledge(k, r, true)
	r = c.Snapshot(k).Review
	ticket, cmd, ok := c.BeginSave(k, r)
	if !ok || !c.MarkSent(ticket) {
		t.Fatal("normalized dispatch")
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("double dispatch")
	}
	old := c.Snapshot(k).Generation
	parent.Bind(RoutineScope{})
	parent.Bind(k)
	if c.ViewCurrent(k, old) || c.Current(ticket) {
		t.Fatal("old view revived")
	}
	c.FinishSave(ticket, pluginrpc.DshbotCapabilityReceipt{Attempted: true}, errors.New("unknown"))
	if !c.Snapshot(k).Uncertain || c.Snapshot(k).Draft != v {
		t.Fatal("sent draft lost")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, textProfileStoreFixture(), true, nil)
	c.Prepare(k)
	if c.Snapshot(k).Review != nil || !c.Snapshot(k).Uncertain {
		t.Fatal("nonmatching readback unlocked")
	}
	fresh := textProfileStoreFixture()
	fresh.Revision++
	fresh.Profile["name"], _ = json.Marshal(cmd.Values.Name)
	fresh.Profile["title"], _ = json.Marshal(cmd.Values.Title)
	fresh.Profile["description"], _ = json.Marshal(cmd.Values.Description)
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, fresh, true, nil)
	if !c.Snapshot(k).Candidate || !c.Adopt(k) || c.Snapshot(k).Uncertain || c.Snapshot(k).Draft != cmd.Values {
		t.Fatal("adoption failed")
	}
	parent.Bind(RoutineScope{HostKey: "other", BotID: "b"})
	if c.Snapshot(RoutineScope{HostKey: "other", BotID: "b"}).Status.Bot.ID != "" {
		t.Fatal("cross-account leak")
	}
}
func TestBotTextProfileReadOnlyDiscardAndUnsent(t *testing.T) {
	var c TextProfileControls
	k := RoutineScope{HostKey: "a", BotID: "b"}
	c.Bind(k)
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, textProfileStoreFixture(), false, nil)
	v := pluginrpc.DshbotTextProfileValues{Name: "Changed"}
	c.SetDraft(k, v)
	if c.Snapshot(k).Dirty {
		t.Fatal("readonly")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, textProfileStoreFixture(), true, nil)
	c.SetDraft(k, v)
	c.Prepare(k)
	stale := c.Snapshot(k).Review
	c.Discard(k)
	if c.Snapshot(k).Dirty || c.Snapshot(k).Review != nil {
		t.Fatal("discard")
	}
	c.SetDraft(k, v)
	c.Prepare(k)
	c.Acknowledge(k, c.Snapshot(k).Review, true)
	if _, _, ok := c.BeginSave(k, stale); ok {
		t.Fatal("old review")
	}
	ticket, _, ok := c.BeginSave(k, c.Snapshot(k).Review)
	if !ok {
		t.Fatal("begin")
	}
	c.Bind(RoutineScope{})
	if c.MarkSent(ticket) {
		t.Fatal("hidden dispatch")
	}
	c.AbortUnsent(ticket, errors.New("view changed"))
	c.Bind(k)
	if c.Snapshot(k).Busy || c.Snapshot(k).Uncertain || c.Snapshot(k).Loaded {
		t.Fatal("unsent recovery")
	}
}
