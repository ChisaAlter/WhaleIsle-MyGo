package dshbot

import (
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func advancedReady(t *testing.T, enabled bool) (*RoutineControls, RoutineScope) {
	t.Helper()
	c := &RoutineControls{}
	k := RoutineScope{HostKey: "fixture-host", BotID: "b"}
	c.Bind(k)
	ticket, ok := c.BeginLoad(k)
	if !ok {
		t.Fatal("load unavailable")
	}
	view := &pluginrpc.DshbotRoutineView{NS: "dsh-bot", Revision: 9}
	view.Value.Items = []pluginrpc.DshbotCatalogItem{{ID: "b", SessionID: "s", WorkspaceID: "/configured/workspace"}}
	view.Value.Routines = []pluginrpc.DshbotRoutine{
		{ID: "r", BotID: "b", Name: "Fixture routine", Prompt: "Fixture prompt", Schedule: "every 1h", Timezone: "UTC", Enabled: enabled, ContextFrom: []string{"self"}, SilentAllowed: true, Watch: json.RawMessage(`{"kind":"command","value":"fixture-only-command","timeoutMs":15000,"future":"keep"}`)},
		{ID: "r2", BotID: "b", Name: "Other routine"}, {ID: "r3", BotID: "b", Name: "Third routine"}, {ID: "r4", BotID: "b", Name: "Fourth routine"}, {ID: "foreign", BotID: "other", Name: "Foreign"},
	}
	if !c.FinishLoad(ticket, view, nil, nil, nil) {
		t.Fatal("load failed")
	}
	c.Select(k, "r")
	c.StartEdit(k)
	return c, k
}
func TestBotRoutineAdvancedUntouchedAndAcknowledgement(t *testing.T) {
	c, k := advancedReady(t, true)
	c.Edit(k, "name", "Edited base name")
	c.Prepare(k, "save")
	s := c.Snapshot(k)
	if s.Review == nil || s.Review.Command.Advanced.WatchSet || s.Review.Command.Advanced.ContextSet || s.Review.Command.Advanced.SilentSet || s.Review.Advanced.WatchValue != "fixture-only-command" || s.Review.WorkspaceID != "/configured/workspace" {
		t.Fatal(s)
	}
	if !RoutineReviewNeedsWatchAcknowledgement(*s.Review) {
		t.Fatal("effective unchanged command hidden")
	}
	if _, _, ok := c.BeginAction(k, s.Review); ok {
		t.Fatal("command enabled without explicit acknowledgement")
	}
	old := s.Review
	c.AcknowledgeWatch(k, old, true)
	c.Edit(k, "watchValue", "fixture-only-new-command")
	c.Prepare(k, "save")
	c.AcknowledgeWatch(k, old, true)
	if c.Snapshot(k).Review.WatchAcknowledged {
		t.Fatal("old acknowledgement consumed new review")
	}
	if _, _, ok := c.BeginAction(k, old); ok {
		t.Fatal("old confirm consumed new review")
	}
	current := c.Snapshot(k).Review
	c.AcknowledgeWatch(k, current, true)
	ticket, cmd, ok := c.BeginAction(k, current)
	if !ok || !cmd.Advanced.WatchSet || cmd.Advanced.Watch.Value != "fixture-only-new-command" || cmd.Advanced.ContextSet {
		t.Fatal(cmd)
	}
	c.FinishAction(ticket, nil, errors.New("synthetic uncertain response"))
	s = c.Snapshot(k)
	if s.Loaded || s.Draft == nil || s.Draft.Advanced.WatchValue != "fixture-only-new-command" || s.Review != nil {
		t.Fatal("uncertain edit lost draft or allowed another action")
	}
}
func TestBotRoutineAdvancedContextOwnershipAndIsolation(t *testing.T) {
	c, k := advancedReady(t, false)
	c.Edit(k, "contextToggle", "foreign")
	if c.Snapshot(k).Error == "" {
		t.Fatal("cross-Bot reference accepted")
	}
	c.Edit(k, "contextToggle", "r")
	if c.Snapshot(k).Error == "" {
		t.Fatal("own ID bypassed self")
	}
	c.Edit(k, "contextToggle", "r2")
	c.Edit(k, "contextToggle", "r3")
	c.Edit(k, "contextToggle", "r4")
	if len(c.Snapshot(k).Draft.Advanced.ContextFrom) != 3 || c.Snapshot(k).Error == "" {
		t.Fatal("context cap bypassed")
	}
	s := c.Snapshot(k)
	s.Draft.Advanced.ContextFrom[0] = "leak"
	c.Edit(k, "silentAllowed", "false")
	c.Prepare(k, "save")
	s = c.Snapshot(k)
	if s.Review == nil || s.Review.Command.Advanced.ContextFrom[0] != "self" || !s.Review.Command.Advanced.SilentSet || s.Review.Command.Advanced.WatchSet {
		t.Fatal(s)
	}
	s.Review.Advanced.ContextFrom[0] = "leak"
	s.Review.Command.Advanced.ContextFrom[0] = "leak"
	if c.Snapshot(k).Review.Command.Advanced.ContextFrom[0] != "self" || c.Snapshot(k).Review.Advanced.ContextFrom[0] != "self" {
		t.Fatal("snapshot aliased review")
	}
	if RoutineReviewNeedsWatchAcknowledgement(*c.Snapshot(k).Review) {
		t.Fatal("paused save unnecessarily gated")
	}
	c.Prepare(k, "toggle")
	if !RoutineReviewNeedsWatchAcknowledgement(*c.Snapshot(k).Review) {
		t.Fatal("enable omitted effective command")
	}
	old := c.Snapshot(k).Review
	c.Bind(RoutineScope{HostKey: "other-host", BotID: "b"})
	c.AcknowledgeWatch(k, old, true)
	if _, _, ok := c.BeginAction(k, old); ok {
		t.Fatal("cross-Host old confirm allowed")
	}
}
func TestBotRoutineAdvancedUnknownPreservedAndExplicitClear(t *testing.T) {
	d := routineAdvancedFromRecord(pluginrpc.DshbotRoutine{ID: "r", BotID: "b", ContextFrom: []string{"missing"}, SilentAllowed: true, Watch: json.RawMessage(`{"kind":"future-kind","opaque":"preserve"}`)})
	rows := []pluginrpc.DshbotRoutine{{ID: "r2", BotID: "b"}}
	p, err := prepareRoutineAdvanced(d, "b", "r", rows)
	if err == nil || p.WatchSet || p.ContextSet || !d.WatchUnknown {
		t.Fatal("unavailable untouched context did not block implicit normalization")
	}
	if _, err = editRoutineAdvanced(&d, "contextToggle", "r2", "b", "r", rows); err != nil {
		t.Fatal(err)
	}
	if _, err = prepareRoutineAdvanced(d, "b", "r", rows); err == nil {
		t.Fatal("missing source silently replaced")
	}
	editRoutineAdvanced(&d, "contextToggle", "missing", "b", "r", rows)
	editRoutineAdvanced(&d, "watchKind", "", "b", "r", rows)
	p, err = prepareRoutineAdvanced(d, "b", "r", rows)
	if err != nil || !p.WatchSet || p.Watch != nil || len(p.ContextFrom) != 1 || p.ContextFrom[0] != "r2" {
		t.Fatal("explicit clear/change incorrect", p, err)
	}
	editRoutineAdvanced(&d, "watchKind", "url", "b", "r", rows)
	editRoutineAdvanced(&d, "watchValue", "https://fixture.invalid", "b", "r", rows)
	editRoutineAdvanced(&d, "watchTimeout", "300001", "b", "r", rows)
	if _, err = prepareRoutineAdvanced(d, "b", "r", rows); err == nil {
		t.Fatal("timeout upper bound bypassed")
	}
}
