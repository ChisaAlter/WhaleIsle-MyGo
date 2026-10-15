package dshbot

import (
	"fmt"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func roomStoreCatalog() pluginrpc.DshbotRoomCatalog {
	return pluginrpc.DshbotRoomCatalog{Revision: 8, Items: []pluginrpc.DshbotRoomItem{{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: "bot", Name: "Alpha", SessionID: "sa"}}, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", Name: "Beta", SessionID: "sb"}}}}
}
func roomStoreReady(t *testing.T) (*RoomCreator, string) {
	c := &RoomCreator{}
	k := "fixture-account"
	c.Bind(k, "client-1")
	ticket, ok := c.BeginLoad(k)
	if !ok {
		t.Fatal("load denied")
	}
	c.FinishLoad(ticket, roomStoreCatalog(), true, nil)
	c.Start(k, "/fixture")
	c.ToggleMember(k, "a")
	c.ToggleMember(k, "b")
	return c, k
}
func roomStoreBegin(t *testing.T, c *RoomCreator, k string) (RoomCreateTicket, pluginrpc.DshbotRoomCreateCommand) {
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("missing impact acknowledgment")
	}
	c.Acknowledge(k, r, true)
	r = c.Snapshot(k).Review
	ticket, cmd, ok := c.Begin(k, r)
	if !ok {
		t.Fatal("explicit creation missing")
	}
	if _, _, ok = c.Begin(k, r); ok {
		t.Fatal("double dispatch")
	}
	if !c.MarkSent(ticket) {
		t.Fatal("not sent")
	}
	return ticket, cmd
}
func TestRoomCreatorUnknownReadbackAndAccountIsolation(t *testing.T) {
	c, k := roomStoreReady(t)
	ticket, cmd := roomStoreBegin(t, c, k)
	c.Bind("other-account", "client-2")
	c.Finish(ticket, pluginrpc.DshbotRoomCreateResult{Attempted: true}, fmt.Errorf("lost response"))
	if c.Snapshot("other-account").Draft != nil {
		t.Fatal("draft leaked")
	}
	c.Bind(k, "client-3")
	s := c.Snapshot(k)
	if !s.Uncertain || s.Draft == nil {
		t.Fatal("lost attempt", s)
	}
	c.Discard(k)
	c.Start(k, "/another")
	if c.Snapshot(k).Draft.ID != cmd.ID {
		t.Fatal("unknown identity replaced")
	}
	load, _ := c.BeginLoad(k)
	newer := roomStoreCatalog()
	newer.Revision++
	c.FinishLoad(load, newer, true, nil)
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("absence unlocked unknown request")
	}
	newer.Items = append(newer.Items, pluginrpc.DshbotRoomItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: cmd.ID, Kind: "room", SessionID: cmd.SessionID, Name: cmd.Draft.Name, MemberBotIDs: cmd.Draft.MemberBotIDs}, MaxRounds: cmd.Draft.MaxRounds, MaxSpeaks: cmd.Draft.MaxSpeaks})
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, newer, true, nil)
	if c.Snapshot(k).Candidate == nil {
		t.Fatal("exact record not offered")
	}
	if !c.Adopt(k, cmd.ID) || c.Snapshot(k).Created.ID != cmd.ID || c.Snapshot(k).Draft != nil {
		t.Fatal("explicit adoption failed")
	}
}
func TestRoomCreatorStaleReviewAndRejectedDraft(t *testing.T) {
	c, k := roomStoreReady(t)
	c.Prepare(k)
	r := c.Snapshot(k).Review
	c.Acknowledge(k, r, true)
	c.Change(k, "name", "Changed")
	if _, _, ok := c.Begin(k, r); ok {
		t.Fatal("stale review dispatched")
	}
	ticket, cmd := roomStoreBegin(t, c, k)
	c.Finish(ticket, pluginrpc.DshbotRoomCreateResult{Attempted: true, Rejected: true}, fmt.Errorf("catalog changed"))
	if s := c.Snapshot(k); s.Draft == nil || s.Draft.ID != cmd.ID || s.Uncertain {
		t.Fatal("rejection lost draft", s)
	}
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("rejection did not require refresh")
	}
	load, _ := c.BeginLoad(k)
	c.Suspend()
	c.Bind(k, "client-1")
	if c.FinishLoad(load, roomStoreCatalog(), true, nil) {
		t.Fatal("stale load accepted")
	}
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, roomStoreCatalog(), false, nil)
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("read-only allowed mutation")
	}
}

func TestRoomCreatorFinalRecoveryKeepsReviewedCwd(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(fmt.Sprint("unknown=", uncertain), func(t *testing.T) {
			c, k := roomStoreReady(t)
			c.Change(k, "scratchCwd", "/reviewed/room-cwd")
			ticket, cmd := roomStoreBegin(t, c, k)
			catalog := roomStoreCatalog()
			catalog.Revision++
			item := pluginrpc.DshbotRoomItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: cmd.ID, Kind: "room", SessionID: cmd.SessionID, Name: cmd.Draft.Name, MemberBotIDs: cmd.Draft.MemberBotIDs}, MaxRounds: cmd.Draft.MaxRounds, MaxSpeaks: cmd.Draft.MaxSpeaks}
			catalog.Items = append(catalog.Items, item)
			if uncertain {
				c.Finish(ticket, pluginrpc.DshbotRoomCreateResult{Attempted: true}, fmt.Errorf("lost response"))
				load, _ := c.BeginLoad(k)
				c.FinishLoad(load, catalog, true, nil)
				if !c.Adopt(k, cmd.ID) {
					t.Fatal("cannot adopt exact record")
				}
			} else {
				c.Finish(ticket, pluginrpc.DshbotRoomCreateResult{Attempted: true, Catalog: catalog, Item: item}, nil)
			}
			c.Suspend()
			c.Bind(k, "reopened-client")
			result := c.Snapshot(k)
			if result.CreatedScratchCwd != "/reviewed/room-cwd" || result.Created.SessionID != cmd.SessionID || result.Draft != nil {
				t.Fatal("recovery lost reviewed cwd/canonical identity", result)
			}
		})
	}
}

func TestRoomCreatorHiddenViewRetiresCallbacksKeepsSentAttempt(t *testing.T) {
	c, k := roomStoreReady(t)
	c.Prepare(k)
	before := c.Snapshot(k)
	c.Acknowledge(k, before.Review, true)
	review := c.Snapshot(k).Review
	c.Suspend()
	c.Bind(k, "client-1")
	if c.ViewCurrent(k, before.Generation) {
		t.Fatal("hidden then reopened view revived old callbacks")
	}
	if _, _, ok := c.Begin(k, review); ok {
		t.Fatal("old hidden-frame confirmation dispatched")
	}
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, roomStoreCatalog(), true, nil)
	ticket, cmd := roomStoreBegin(t, c, k)
	sentView := c.Snapshot(k).Generation
	c.Suspend()
	c.Bind(k, "client-1")
	if c.ViewCurrent(k, sentView) || c.Current(ticket) {
		t.Fatal("late opened-view callback still current")
	}
	c.Finish(ticket, pluginrpc.DshbotRoomCreateResult{Attempted: true}, fmt.Errorf("response lost after hide"))
	result := c.Snapshot(k)
	if !result.Uncertain || result.Busy || result.Draft == nil || result.Draft.ID != cmd.ID {
		t.Fatal("sent origin attempt was discarded by lifecycle retirement", result)
	}
}
