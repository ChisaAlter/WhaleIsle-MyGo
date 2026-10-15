package dshbot

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"testing"
	"whaleisle/internal/desktop/avatarimage"
	"whaleisle/internal/desktop/pluginrpc"
)

func imageStoreFixture() pluginrpc.DshbotCapabilityStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(`{"id":"b","kind":"bot","sessionId":"s","name":"Fixture","title":"","sectionId":"","description":"Keep","workspaceId":"/fixture","avatar":{"kind":"blob","shape":"circle","color":"blue"},"model":{"provider":"","model":"","reasoningEffort":""},"pinned":false,"hidden":false,"pinOrder":0,"allowedSenderIds":[],"memberBotIds":[],"objective":"Existing objective","objectiveMaxRounds":16,"memoryReview":false,"notify":true,"capabilities":{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"selected","names":[]}},"maxRounds":3,"maxSpeaks":10,"inbox":[],"readInitialized":true,"lastSeenSeq":0,"createdAt":1,"updatedAt":2}`), &p)
	var policy pluginrpc.DshbotCapabilityPolicies
	json.Unmarshal(p["capabilities"], &policy)
	return pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 2, Profile: p, Policies: policy, Catalog: pluginrpc.DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []pluginrpc.DshbotCapabilityEntry{}, Skills: []pluginrpc.DshbotCapabilityEntry{}, MCP: []pluginrpc.DshbotCapabilityEntry{}}}
}

func TestBotAvatarImageImportLifecycleAndUnknown(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 40, 20)))
	source, err := avatarimage.Decode(b.Bytes(), "synthetic.png", "png")
	if err != nil {
		t.Fatal(err)
	}
	var parent RoutineControls
	k := RoutineScope{HostKey: "fake-account", BotID: "b"}
	parent.Bind(k)
	c := &parent.Avatar
	load := func() {
		ticket, ok := c.BeginLoad(k)
		if !ok {
			t.Fatal("load blocked")
		}
		if !c.FinishLoad(ticket, imageStoreFixture(), true, nil) {
			t.Fatal("load failed")
		}
	}
	load()
	original := c.Snapshot(k).Draft
	ticket, ok := c.BeginImageImport(k)
	if !ok || !c.ImageImportCurrent(ticket) {
		t.Fatal("begin import")
	}
	if _, ok := c.BeginLoad(k); ok {
		t.Fatal("refresh during import")
	}
	c.SetDraft(k, pluginrpc.DshbotAvatarValues{Kind: "blob", Shape: "hex", Color: "red"})
	if c.Snapshot(k).Draft != original {
		t.Fatal("changed during chooser")
	}
	if !c.FinishImageImport(ticket, source, nil) || !c.Snapshot(k).ImageImport.Ready || c.Snapshot(k).Draft != original {
		t.Fatal("import silently replaced draft")
	}
	crop := avatarimage.DefaultCrop("square")
	crop.Zoom = 2
	crop.X = 1
	c.UpdateImageCrop(k, crop)
	if c.Snapshot(k).ImageImport.Crop != crop {
		t.Fatal("crop not consumed")
	}
	c.CancelImageImport(k)
	if c.imageImport != nil || c.Snapshot(k).ImageImport.Ready || c.Snapshot(k).Draft != original {
		t.Fatal("cancel retained source")
	}
	ticket, _ = c.BeginImageImport(k)
	parent.Bind(RoutineScope{})
	parent.Bind(k)
	if c.FinishImageImport(ticket, source, nil) || c.ImageImportCurrent(ticket) || c.imageImport != nil || c.Snapshot(k).ImageImport.Busy {
		t.Fatal("hidden selection revived")
	}
	load()
	ticket, _ = c.BeginImageImport(k)
	c.FinishImageImport(ticket, source, nil)
	if !c.ApplyImageCrop(k) {
		t.Fatal("apply")
	}
	applied := c.Snapshot(k).Draft
	if applied.Kind != "image" || !c.Snapshot(k).Dirty || c.imageImport != nil {
		t.Fatal("apply/source release")
	}
	parent.Bind(RoutineScope{})
	parent.Bind(k)
	if c.Snapshot(k).Draft != original || c.Snapshot(k).Dirty {
		t.Fatal("hidden unsent image retained")
	}
	load()
	ticket, _ = c.BeginImageImport(k)
	c.FinishImageImport(ticket, source, nil)
	c.ApplyImageCrop(k)
	c.Prepare(k)
	review := c.Snapshot(k).Review
	if review == nil {
		t.Fatal("image review", c.Snapshot(k).Error)
	}
	if _, _, ok := c.BeginSave(k, review); ok {
		t.Fatal("missing impact acknowledgement")
	}
	c.Acknowledge(k, review, true)
	save, cmd, ok := c.BeginSave(k, c.Snapshot(k).Review)
	if !ok || !c.MarkSent(save) {
		t.Fatal("save")
	}
	parent.Bind(RoutineScope{})
	parent.Bind(RoutineScope{HostKey: "other-account", BotID: "b"})
	c.FinishSave(save, pluginrpc.DshbotCapabilityReceipt{Attempted: true}, errors.New("unknown"))
	parent.Bind(k)
	if !c.Snapshot(k).Uncertain || c.Snapshot(k).Draft != applied || c.imageImport != nil {
		t.Fatal("unknown image draft lost")
	}
	load()
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("unknown resent")
	}
	fresh := imageStoreFixture()
	fresh.Revision++
	fresh.Profile["avatar"], _ = json.Marshal(cmd.Values)
	read, _ := c.BeginLoad(k)
	c.FinishLoad(read, fresh, true, nil)
	if !c.Snapshot(k).Candidate || !c.Adopt(k) {
		t.Fatal("exact image readback")
	}
}
