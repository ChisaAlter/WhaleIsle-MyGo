package dshbot

import (
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func memoryFixture() pluginrpc.DshbotMemoryResult {
	return pluginrpc.DshbotMemoryResult{BotID: "b", Bot: pluginrpc.DshbotMemoryTrack{Entries: []string{"old"}, Text: "- old", Revision: "bot-v1"}, User: pluginrpc.DshbotMemoryTrack{Entries: []string{"user"}, Text: "- user", Revision: "user-v1"}, Limits: pluginrpc.DshbotMemoryLimits{Bot: 8000, User: 4000}}
}
func memoryLoaded(t *testing.T) (*MemoryControls, RoutineScope) {
	t.Helper()
	c := &MemoryControls{}
	k := RoutineScope{HostKey: "fixture-auth", BotID: "b"}
	c.Bind(k)
	ticket, _ := c.BeginLoad(k)
	if !c.FinishLoad(ticket, pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s", Name: "Bot"}, memoryFixture(), true, nil) {
		t.Fatal("load")
	}
	return c, k
}
func memoryPrepared(t *testing.T, c *MemoryControls, k RoutineScope) *MemoryReview {
	t.Helper()
	c.StartEdit(k, 0)
	c.Change(k, "edited")
	c.Prepare(k)
	r := c.Snapshot(k).Review
	if r == nil {
		t.Fatal("review missing", c.Snapshot(k).Error)
	}
	return r
}
func TestBotMemoryCASDraftAndUnknownResult(t *testing.T) {
	c, k := memoryLoaded(t)
	oldView := c.Snapshot(k).Generation
	r := memoryPrepared(t, c, k)
	if c.ViewCurrent(k, oldView) {
		t.Fatal("old rendered memory callbacks survived new draft")
	}
	ticket, cmd, ok := c.BeginSave(k, r)
	if !ok || cmd.MemoryRevision != "bot-v1" || cmd.Text != "- edited" || cmd.Track != "bot" {
		t.Fatal(cmd)
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("duplicate save")
	}
	c.FinishSave(ticket, pluginrpc.DshbotMemoryReceipt{}, errors.New("synthetic uncertainty"))
	s := c.Snapshot(k)
	if s.Loaded || s.Draft == nil || s.Draft.Input != "edited" {
		t.Fatal("unknown result lost draft or enabled retry")
	}
	load, _ := c.BeginLoad(k)
	fresh := memoryFixture()
	fresh.Bot.Text = "- edited"
	fresh.Bot.Entries = []string{"edited"}
	fresh.Bot.Revision = "bot-v2"
	c.FinishLoad(load, pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s"}, fresh, true, nil)
	if c.Snapshot(k).Draft != nil || !c.Snapshot(k).Loaded || c.Snapshot(k).Server.User.Revision != "user-v1" {
		t.Fatal("readback did not reconcile exact content")
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("old confirm replayed")
	}
}
func TestBotMemoryConflictScopeAndReadonly(t *testing.T) {
	c, k := memoryLoaded(t)
	r := memoryPrepared(t, c, k)
	load, _ := c.BeginLoad(k)
	fresh := memoryFixture()
	fresh.Bot.Revision = "external-v2"
	fresh.Bot.Text = "- other"
	fresh.Bot.Entries = []string{"other"}
	c.FinishLoad(load, pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s"}, fresh, true, nil)
	c.Prepare(k)
	if c.Snapshot(k).Review != nil || c.Snapshot(k).Draft.Input != "edited" {
		t.Fatal("silently rebased conflicting draft")
	}
	c.Discard(k)
	c.SetTrack(k, "user")
	c.StartEdit(k, 0)
	if c.Snapshot(k).Draft.BaseRevision != "user-v1" {
		t.Fatal("track CAS mixed")
	}
	snap := c.Snapshot(k)
	snap.Server.User.Entries[0] = "mutated"
	snap.Draft.Entries[0] = "mutated"
	if c.Snapshot(k).Server.User.Entries[0] != "user" || c.Snapshot(k).Draft.Entries[0] != "user" {
		t.Fatal("snapshot aliases")
	}
	c.Discard(k)
	load, _ = c.BeginLoad(k)
	c.FinishLoad(load, pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s"}, memoryFixture(), false, nil)
	c.StartEdit(k, 0)
	if c.Snapshot(k).Draft != nil {
		t.Fatal("readonly edited")
	}
	c.Bind(RoutineScope{})
	c.Bind(k)
	if c.FinishLoad(load, pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s"}, memoryFixture(), true, nil) {
		t.Fatal("closed/reopened late read applied")
	}
	if _, _, ok := c.BeginSave(k, r); ok {
		t.Fatal("foreign review")
	}
}
func TestBotMemoryDeleteAndLifecycleFence(t *testing.T) {
	var owner RoutineControls
	k := RoutineScope{HostKey: "a", BotID: "b"}
	owner.Bind(k)
	c := &owner.Memory
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s"}, memoryFixture(), true, nil)
	c.PrepareDelete(k, 0)
	r := c.Snapshot(k).Review
	if r == nil || r.Command.Text != "" || r.Before != "- old" {
		t.Fatal("delete must review whole-track empty replacement")
	}
	ticket, _, ok := c.BeginSave(k, r)
	if !ok || !c.Current(ticket) {
		t.Fatal("confirmed delete")
	}
	owner.Bind(RoutineScope{})
	owner.Bind(k)
	if c.Current(ticket) || c.Snapshot(k).Draft != nil {
		t.Fatal("parent lifecycle retained memory")
	}
}

func TestBotMemorySuccessfulReceiptAndStaleReview(t *testing.T) {
	c, k := memoryLoaded(t)
	old := memoryPrepared(t, c, k)
	c.Change(k, "final")
	c.Prepare(k)
	fresh := c.Snapshot(k).Review
	if _, _, ok := c.BeginSave(k, old); ok {
		t.Fatal("old confirm consumed replacement draft")
	}
	ticket, cmd, ok := c.BeginSave(k, fresh)
	if !ok {
		t.Fatal("fresh review rejected")
	}
	out := pluginrpc.DshbotMemoryReceipt{BotID: "b", Track: "bot", MemoryRevision: "bot-v2", DshbotMemoryTrack: pluginrpc.DshbotMemoryTrack{Text: cmd.Text, Entries: []string{"final"}, Revision: "bot-v2"}, View: pluginrpc.DshbotSettingsView{NS: "dsh-bot", Value: pluginrpc.DshbotCatalogValue{Items: []pluginrpc.DshbotCatalogItem{{ID: "b", SessionID: "s"}}}}}
	if !c.FinishSave(ticket, out, nil) {
		t.Fatal("success missing")
	}
	s := c.Snapshot(k)
	if !s.Loaded || s.Draft != nil || s.Server.Bot.Text != "- final" || s.Server.User.Text != "- user" || c.Current(ticket) {
		t.Fatal("receipt changed wrong track or stayed pending")
	}
	if c.FinishSave(ticket, out, nil) {
		t.Fatal("duplicate completion applied")
	}
}
