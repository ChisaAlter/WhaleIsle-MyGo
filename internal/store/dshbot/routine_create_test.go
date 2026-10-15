package dshbot

import (
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func createSource(rev int) RoutineSnapshot {
	return RoutineSnapshot{Scope: RoutineScope{HostKey: "client-1", BotID: "b"}, Loaded: true, Revision: rev, BotName: "Bot B", SessionID: "s"}
}
func createReady(t *testing.T, c *RoutineCreator, key RoutineScope) RoutineCreateSnapshot {
	t.Helper()
	c.Sync(key, 1, createSource(4))
	c.Start(key, "Asia/Shanghai")
	c.Change(key, "prompt", "Fixture prompt")
	c.Prepare(key)
	s := c.Snapshot(key)
	if s.Review == nil {
		t.Fatal(s.Error)
	}
	return s
}
func TestBotRoutineCreateDefaultReviewAndEpoch(t *testing.T) {
	var c RoutineCreator
	k := RoutineScope{HostKey: "account", BotID: "b"}
	c.Sync(k, 1, createSource(4))
	c.Start(k, "Asia/Shanghai")
	s := c.Snapshot(k)
	if s.Draft.Hour != "09" || s.Draft.Minute != "00" || s.Draft.Enabled || s.Draft.MaxRuns != "0" {
		t.Fatal(s.Draft)
	}
	if _, _, ok := c.Begin(k, c.Snapshot(k).Review); ok {
		t.Fatal("created without review")
	}
	c.Change(k, "prompt", "Fixture prompt")
	c.Prepare(k)
	ticket, cmd, ok := c.Begin(k, c.Snapshot(k).Review)
	if !ok || cmd.Edit.Schedule != "0 9 * * *" || cmd.RequestID == "" {
		t.Fatal(cmd)
	}
	if _, _, ok := c.Begin(k, c.Snapshot(k).Review); ok {
		t.Fatal("double submit")
	}
	c.Sync(k, 2, createSource(4))
	if c.Current(ticket) {
		t.Fatal("old dispatch survived close/reopen epoch")
	}
	c.AbortUnsent(ticket)
	c.Prepare(k)
	ticket, _, ok = c.Begin(k, c.Snapshot(k).Review)
	if !ok {
		t.Fatal(c.Snapshot(k).Error)
	}
	c.MarkSent(ticket)
	c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{}, errors.New("lost response"))
	if c.Snapshot(k).Draft.Prompt != "Fixture prompt" || !c.Snapshot(k).Uncertain {
		t.Fatal("lost draft or uncertain marker")
	}
	c.Change(k, "prompt", "Changed")
	if c.Snapshot(k).Draft.Prompt != "Fixture prompt" {
		t.Fatal("uncertain sent identity edited")
	}
	c.Prepare(k)
	if _, _, ok := c.Begin(k, c.Snapshot(k).Review); ok {
		t.Fatal("uncertain create replayed")
	}
}
func TestBotRoutineCreateReadbackAndSafeRebase(t *testing.T) {
	var c RoutineCreator
	k := RoutineScope{HostKey: "account", BotID: "b"}
	createReady(t, &c, k)
	ticket, cmd, _ := c.Begin(k, c.Snapshot(k).Review)
	c.MarkSent(ticket)
	c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{}, errors.New("lost"))
	c.Sync(k, 1, createSource(4))
	c.Rebase(k)
	if !c.Snapshot(k).Uncertain {
		t.Fatal("same-revision read unlocked replay")
	}
	s := createSource(5)
	s.Routines = []pluginrpc.DshbotRoutine{{ID: "new", BotID: "b", Name: cmd.Edit.Name, Prompt: cmd.Edit.Prompt, Schedule: cmd.Edit.Schedule, Timezone: cmd.Edit.Timezone}}
	c.Sync(k, 1, s)
	if len(c.Snapshot(k).Candidates) != 1 || c.Snapshot(k).RetryVerified {
		t.Fatal("new row not quarantined")
	}
	if !c.Adopt(k, "new") {
		t.Fatal("explicit adoption failed")
	}
	if c.Snapshot(k).Draft != nil || c.Snapshot(k).Uncertain {
		t.Fatal("adoption did not resolve draft")
	}
	c.Start(k, "Asia/Shanghai")
	c.Change(k, "prompt", "Second prompt")
	c.Prepare(k)
	ticket, _, _ = c.Begin(k, c.Snapshot(k).Review)
	c.MarkSent(ticket)
	c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{}, errors.New("lost"))
	s.Revision = 6
	c.Sync(k, 1, s)
	if !c.Snapshot(k).RetryVerified {
		t.Fatal("newer revision with no new record did not fence old CAS")
	}
	c.Rebase(k)
	if c.Snapshot(k).Uncertain || c.Snapshot(k).Draft.Prompt != "Second prompt" || c.Snapshot(k).Draft.Revision != 6 {
		t.Fatal(c.Snapshot(k))
	}
}
func TestBotRoutineCreateCrossHostAndReturnedIdentity(t *testing.T) {
	var c RoutineCreator
	k := RoutineScope{HostKey: "a", BotID: "b"}
	createReady(t, &c, k)
	ticket, cmd, _ := c.Begin(k, c.Snapshot(k).Review)
	c.MarkSent(ticket)
	other := RoutineScope{HostKey: "b", BotID: "b"}
	c.Sync(other, 3, createSource(8))
	if c.Current(ticket) {
		t.Fatal("foreign Host still current")
	}
	v := &pluginrpc.DshbotRoutineView{NS: "dsh-bot", Revision: 5}
	v.Value.Items = []pluginrpc.DshbotCatalogItem{{ID: "b", SessionID: "s"}}
	v.Value.Routines = []pluginrpc.DshbotRoutine{{ID: "created", BotID: "b", Name: cmd.Edit.Name, Prompt: cmd.Edit.Prompt, Schedule: cmd.Edit.Schedule, Timezone: cmd.Edit.Timezone, SilentAllowed: true}}
	c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{ID: "created", View: v}, nil)
	if c.Snapshot(other).CreatedID != "" {
		t.Fatal("foreign result displayed")
	}
	c.Sync(k, 4, createSource(5))
	if c.Snapshot(k).CreatedID != "created" || c.Snapshot(k).Busy {
		t.Fatal("original result lost on navigation")
	}
}

func TestBotRoutineCreateRejectedDraftRemainsEditable(t *testing.T) {
	var c RoutineCreator
	k := RoutineScope{HostKey: "account", BotID: "b"}
	createReady(t, &c, k)
	oldReview := c.Snapshot(k).Review
	c.Change(k, "name", "Changed")
	c.Prepare(k)
	if _, _, ok := c.Begin(k, oldReview); ok {
		t.Fatal("old confirmation consumed newer draft")
	}
	ticket, _, ok := c.Begin(k, c.Snapshot(k).Review)
	if !ok {
		t.Fatal("review unavailable")
	}
	c.MarkSent(ticket)
	c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{Rejected: true}, errors.New("Host rejected before write"))
	c.Change(k, "name", "Corrected")
	if c.Snapshot(k).Uncertain || c.Snapshot(k).Draft.Name != "Corrected" {
		t.Fatal("known rejection locked draft")
	}
	if RoutineCreateTimezone("Local") != "" || RoutineCreateTimezone("CST") != "" || RoutineCreateTimezone("Asia/Shanghai") != "Asia/Shanghai" {
		t.Fatal("timezone guessed")
	}
}
