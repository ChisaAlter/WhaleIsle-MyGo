package dshbot

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func notifyStoreFixture() pluginrpc.DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"","sectionId":"","description":"Keep","workspaceId":"/fixture","avatar":{"kind":"blob","shape":"circle","color":"blue"},"model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":false,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2}`), &p)
	var policy pluginrpc.DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	return pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 2, Profile: p, Policies: policy, Catalog: pluginrpc.DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []pluginrpc.DshbotCapabilityEntry{}, Skills: []pluginrpc.DshbotCapabilityEntry{}, MCP: []pluginrpc.DshbotCapabilityEntry{}}}
}

func TestBotNotifyLifecycleUnknownAndExactAdoption(t *testing.T) {
	var parent RoutineControls
	k := RoutineScope{HostKey: "account-a", BotID: "b"}
	parent.Bind(k)
	c := &parent.Notify
	ticket, _ := c.BeginLoad(k)
	c.FinishLoad(ticket, notifyStoreFixture(), false, nil)
	c.SetDraft(k, true)
	if c.Snapshot(k).Dirty {
		t.Fatal("readonly")
	}
	ticket, _ = c.BeginLoad(k)
	c.FinishLoad(ticket, notifyStoreFixture(), true, nil)
	c.SetDraft(k, true)
	c.Prepare(k)
	review := c.Snapshot(k).Review
	if review == nil || !review.Command.Enabled {
		t.Fatal("review")
	}
	if _, _, ok := c.BeginSave(k, review); ok {
		t.Fatal("unacknowledged")
	}
	c.Acknowledge(k, review, true)
	save, cmd, ok := c.BeginSave(k, c.Snapshot(k).Review)
	if !ok || !c.MarkSent(save) {
		t.Fatal("dispatch")
	}
	if c.MarkSent(save) {
		t.Fatal("double dispatch")
	}
	generation := c.Snapshot(k).Generation
	parent.Bind(RoutineScope{})
	parent.Bind(RoutineScope{HostKey: "account-b", BotID: "b"})
	c.FinishSave(save, pluginrpc.DshbotCapabilityReceipt{Attempted: true}, errors.New("unknown"))
	parent.Bind(k)
	if c.ViewCurrent(k, generation) || c.Current(save) || !c.Snapshot(k).Uncertain || !c.Snapshot(k).Draft {
		t.Fatal("hidden original attempt lost")
	}
	ticket, _ = c.BeginLoad(k)
	c.FinishLoad(ticket, notifyStoreFixture(), true, nil)
	c.Prepare(k)
	c.Discard(k)
	if c.Snapshot(k).Review != nil || !c.Snapshot(k).Uncertain || !c.Snapshot(k).Draft {
		t.Fatal("unknown unlocked")
	}
	fresh := notifyStoreFixture()
	fresh.Revision++
	fresh.Profile["notify"], _ = json.Marshal(cmd.Enabled)
	bad := pluginrpc.CloneDshbotCapabilities(fresh)
	bad.Profile["hidden"] = json.RawMessage(`true`)
	ticket, _ = c.BeginLoad(k)
	c.FinishLoad(ticket, bad, true, nil)
	if c.Snapshot(k).Candidate {
		t.Fatal("unrelated hidden change treated success")
	}
	ticket, _ = c.BeginLoad(k)
	c.FinishLoad(ticket, fresh, true, nil)
	if !c.Snapshot(k).Candidate || !c.Adopt(k) || c.Snapshot(k).Uncertain {
		t.Fatal("exact adoption")
	}
	c.SetDraft(k, false)
	c.Prepare(k)
	stale := c.Snapshot(k).Review
	c.Discard(k)
	if c.Snapshot(k).Dirty || c.Snapshot(k).Review != nil {
		t.Fatal("discard")
	}
	c.SetDraft(k, false)
	c.Prepare(k)
	c.Acknowledge(k, c.Snapshot(k).Review, true)
	if _, _, ok := c.BeginSave(k, stale); ok {
		t.Fatal("old confirmation")
	}
	unsent, _, ok := c.BeginSave(k, c.Snapshot(k).Review)
	if !ok {
		t.Fatal("unsent begin")
	}
	parent.Bind(RoutineScope{})
	if c.MarkSent(unsent) {
		t.Fatal("hidden dispatch")
	}
	c.AbortUnsent(unsent, errors.New("view changed"))
	parent.Bind(k)
	if c.Snapshot(k).Busy || c.Snapshot(k).Uncertain {
		t.Fatal("unsent recovery")
	}
}
