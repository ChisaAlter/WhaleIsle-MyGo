package dshbot

import (
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func routineView(rev int) *pluginrpc.DshbotRoutineView {
	v := &pluginrpc.DshbotRoutineView{NS: "dsh-bot", Revision: rev}
	v.Value.Items = []pluginrpc.DshbotCatalogItem{{ID: "b", Name: "Bot B", SessionID: "s"}}
	v.Value.Routines = []pluginrpc.DshbotRoutine{{ID: "r", BotID: "b", Name: "Original", Prompt: "Fixture prompt", Schedule: "every 1h", Timezone: "UTC", Enabled: true}, {ID: "r2", BotID: "b", Name: "Second", Prompt: "Second prompt", Schedule: "30m", Timezone: "UTC"}, {ID: "other", BotID: "other-bot"}}
	return v
}
func loadRoutines(t *testing.T, c *RoutineControls, s RoutineScope, rev int) {
	t.Helper()
	ticket, ok := c.BeginLoad(s)
	if !ok {
		t.Fatal("load blocked")
	}
	if !c.FinishLoad(ticket, routineView(rev), nil, nil, nil) {
		t.Fatal("load dropped")
	}
}
func TestBotRoutineScopeDraftRevisionAndConfirmation(t *testing.T) {
	var c RoutineControls
	s := RoutineScope{HostKey: "host-a", BotID: "b"}
	c.Bind(s)
	loadRoutines(t, &c, s, 4)
	if len(c.Snapshot(s).Routines) != 2 {
		t.Fatal("foreign routine exposed")
	}
	c.Select(s, "r")
	c.StartEdit(s)
	c.Edit(s, "name", "Changed")
	c.Select(s, "r2")
	c.StartEdit(s)
	c.Edit(s, "name", "Second draft")
	c.Select(s, "r")
	if c.Snapshot(s).Draft.Edit.Name != "Changed" {
		t.Fatal("routine drafts mixed")
	}
	if _, _, ok := c.BeginAction(s); ok {
		t.Fatal("unreviewed action")
	}
	c.Prepare(s, "save")
	if c.Snapshot(s).Review == nil {
		t.Fatal("missing review")
	}
	c.Edit(s, "prompt", "New prompt")
	if c.Snapshot(s).Review != nil {
		t.Fatal("edited draft kept prior approval")
	}
	c.Prepare(s, "save")
	loadRoutines(t, &c, s, 5)
	snap := c.Snapshot(s)
	if snap.Draft.Edit.Name != "Changed" || snap.Draft.Revision != 4 || snap.Review != nil {
		t.Fatal("refresh rebased/discarded draft or approval")
	}
	c.Prepare(s, "save")
	if _, _, ok := c.BeginAction(s); ok {
		t.Fatal("stale draft can save")
	}
	c.DiscardEdit(s)
	c.StartEdit(s)
	c.Edit(s, "name", "Current")
	c.Prepare(s, "save")
	ticket, _, ok := c.BeginAction(s)
	if !ok {
		t.Fatal(c.Snapshot(s).Error)
	}
	c.Bind(RoutineScope{HostKey: "host-b", BotID: "b"})
	if c.FinishAction(ticket, routineView(6), nil) {
		t.Fatal("late cross-Host write result applied")
	}
	c.Bind(s)
	if c.Snapshot(s).Draft != nil || c.Snapshot(s).Loaded {
		t.Fatal("old Host draft revived")
	}
}
func TestBotRoutineAmbiguousWriteAndCanonicalIdentity(t *testing.T) {
	var c RoutineControls
	s := RoutineScope{HostKey: "h", BotID: "b"}
	c.Bind(s)
	loadRoutines(t, &c, s, 4)
	c.Select(s, "r")
	c.Prepare(s, "run")
	ticket, _, ok := c.BeginAction(s)
	if !ok {
		t.Fatal("not ready")
	}
	if _, _, ok := c.BeginAction(s); ok {
		t.Fatal("double click allowed")
	}
	c.FinishAction(ticket, nil, errors.New("ambiguous transport"))
	if c.Snapshot(s).Loaded || c.Snapshot(s).Review != nil {
		t.Fatal("ambiguous write permits replay")
	}
	c.Prepare(s, "run")
	if _, _, ok := c.BeginAction(s); ok {
		t.Fatal("ambiguous action repeated")
	}
	loadRoutines(t, &c, s, 5)
	c.Prepare(s, "delete")
	ticket, _, ok = c.BeginAction(s)
	if !ok {
		t.Fatal("review unavailable")
	}
	v := routineView(6)
	v.Value.Routines = v.Value.Routines[1:]
	v.Value.Items[0].SessionID = "replaced"
	c.FinishAction(ticket, v, nil)
	if c.Snapshot(s).Loaded || c.Snapshot(s).Error == "" {
		t.Fatal("canonical session changed silently")
	}
}
func TestBotRoutineSnapshotsAndReadFailure(t *testing.T) {
	var c RoutineControls
	s := RoutineScope{HostKey: "h", BotID: "b"}
	c.Bind(s)
	loadRoutines(t, &c, s, 4)
	c.Select(s, "r")
	c.StartEdit(s)
	c.Edit(s, "name", "Keep draft")
	snap := c.Snapshot(s)
	snap.Draft.Edit.Name = "External mutation"
	snap.Routines[0].Name = "External row"
	if c.Snapshot(s).Draft.Edit.Name != "Keep draft" || c.Snapshot(s).Routines[0].Name == "External row" {
		t.Fatal("snapshot aliases state")
	}
	ticket, _ := c.BeginLoad(s)
	c.FinishLoad(ticket, nil, nil, nil, errors.New("read failed"))
	snap = c.Snapshot(s)
	if snap.Loaded || snap.Draft.Edit.Name != "Keep draft" {
		t.Fatal("read failure discarded draft or left writes enabled")
	}
}
