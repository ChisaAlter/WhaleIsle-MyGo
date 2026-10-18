package dshbot

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func senderPolicyStoreFixture() pluginrpc.DshbotSenderPolicyStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"","sectionId":"","description":"Keep","workspaceId":"w1","avatar":{"kind":"blob","shape":"circle","color":"blue"},"model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2}`), &p)
	var policies pluginrpc.DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policies)
	raw, _ := json.Marshal(p)
	return pluginrpc.DshbotSenderPolicyStatus{DshbotCapabilityStatus: pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 8, Profile: p, Policies: policies, PeerIDs: []string{"peer-a", "peer-b"}}, CatalogItems: []json.RawMessage{raw, json.RawMessage(`{"id":"peer-a","kind":"bot","name":"Peer A","sessionId":"a"}`), json.RawMessage(`{"id":"peer-b","kind":"bot","name":"Peer B","sessionId":"pb"}`), json.RawMessage(`{"id":"room","kind":"room","memberBotIds":["b","peer-a"]}`)}, Peers: []pluginrpc.DshbotCatalogItem{{ID: "peer-a", Kind: "bot", Name: "Peer A"}, {ID: "peer-b", Kind: "bot", Name: "Peer B"}}, SharedRoomPeerIDs: []string{"peer-a"}}
}
func TestBotSenderPolicyLifecycleAcknowledgementAndUnknownReadback(t *testing.T) {
	var parent RoutineControls
	k := RoutineScope{HostKey: "fixture-account", BotID: "b"}
	parent.Bind(k)
	c := &parent.SenderPolicy
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, senderPolicyStoreFixture(), true, nil)
	v := c.Snapshot(k).Draft
	v = pluginrpc.DshbotSenderPolicyValues{Selected: true, IDs: []string{"peer-b"}}
	c.SetDraft(k, v)
	snapshot := c.Snapshot(k)
	snapshot.Draft.IDs[0] = "forged"
	snapshot.Status.CatalogItems[1] = json.RawMessage(`{}`)
	if c.Snapshot(k).Draft.IDs[0] != "peer-b" {
		t.Fatal("draft alias escaped")
	}
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("missing independent acknowledgement")
	}
	c.Acknowledge(k, r, true)
	if _, _, ok := c.BeginSave(k, c.Snapshot(k).Review); ok {
		t.Fatal("expansion lacks explicit approval")
	}
	c.AcknowledgeExpansion(k, c.Snapshot(k).Review, true)
	r = c.Snapshot(k).Review
	ticket, cmd, ok := c.BeginSave(k, r)
	if !ok || !c.MarkSent(ticket) {
		t.Fatal("dispatch")
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("double dispatch")
	}
	old := c.Snapshot(k).Generation
	parent.Bind(RoutineScope{})
	parent.Bind(k)
	if c.ViewCurrent(k, old) || c.Current(ticket) {
		t.Fatal("retired view revived")
	}
	c.FinishSave(ticket, pluginrpc.DshbotCapabilityReceipt{Attempted: true}, errors.New("unknown"))
	if !c.Snapshot(k).Uncertain || !pluginrpc.SameDshbotSenderPolicyValues(c.Snapshot(k).Draft, v) {
		t.Fatal("old scope receipt lost")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, senderPolicyStoreFixture(), true, nil)
	c.Prepare(k)
	if c.Snapshot(k).Review != nil || !c.Snapshot(k).Uncertain {
		t.Fatal("unmatched readback released request")
	}
	fresh := senderPolicyStoreFixture()
	fresh.Revision++
	fresh.Profile["allowedSenderIds"] = json.RawMessage(`["peer-b"]`)
	fresh.CatalogItems[0], _ = json.Marshal(fresh.Profile)
	if err := pluginrpc.DshbotSenderPolicyApplied(cmd, fresh.DshbotCapabilityStatus); err != nil {
		t.Fatal(err)
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, fresh, true, nil)
	if !c.Snapshot(k).Candidate {
		t.Fatal("matching not offered")
	}
	if !c.Adopt(k) || c.Snapshot(k).Uncertain || c.Snapshot(k).Dirty {
		t.Fatal("adoption failed")
	}
	parent.Bind(RoutineScope{HostKey: "other", BotID: "b"})
	if c.Snapshot(RoutineScope{HostKey: "other", BotID: "b"}).Status.Bot.ID != "" {
		t.Fatal("cross-account leak")
	}
}
func TestBotSenderPolicyUnsentAndReadOnlyGuards(t *testing.T) {
	var c SenderPolicyControls
	k := RoutineScope{HostKey: "a", BotID: "b"}
	c.Bind(k)
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, senderPolicyStoreFixture(), false, nil)
	v := c.Snapshot(k).Draft
	v = pluginrpc.DshbotSenderPolicyValues{Selected: true, IDs: []string{"peer-b"}}
	c.SetDraft(k, v)
	if c.Snapshot(k).Dirty {
		t.Fatal("readonly")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, senderPolicyStoreFixture(), true, nil)
	c.SetDraft(k, v)
	c.Prepare(k)
	c.Acknowledge(k, c.Snapshot(k).Review, true)
	c.AcknowledgeExpansion(k, c.Snapshot(k).Review, true)
	ticket, _, ok := c.BeginSave(k, c.Snapshot(k).Review)
	if !ok {
		t.Fatal("begin")
	}
	c.Bind(RoutineScope{})
	if c.MarkSent(ticket) {
		t.Fatal("sent from hidden view")
	}
	c.AbortUnsent(ticket, errors.New("view changed"))
	c.Bind(k)
	if c.Snapshot(k).Busy || c.Snapshot(k).Uncertain || c.Snapshot(k).Loaded {
		t.Fatal("unsent recovery")
	}
}
