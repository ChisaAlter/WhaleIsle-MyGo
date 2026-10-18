package dshbot

import (
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func taskControlFixture(revision int, status string) *pluginrpc.DshbotTaskView {
	v := &pluginrpc.DshbotTaskView{NS: "dsh-bot", Revision: revision}
	v.Value.Items = []pluginrpc.DshbotCatalogItem{{ID: "b", Name: "Bot B", SessionID: "s-b"}}
	v.Value.Tasks = []pluginrpc.DshbotTask{{ID: "t", FromID: "a", ToID: "b", Status: status, Task: "Synthetic fixture"}, {ID: "foreign", FromID: "x", ToID: "y", Status: "queued"}}
	return v
}
func loadTaskControl(t *testing.T, c *TaskControls, scope TaskScope, revision int, status string) {
	t.Helper()
	c.Bind(scope)
	ticket, ok := c.BeginLoad(scope)
	if !ok {
		t.Fatal("load denied")
	}
	if !c.FinishLoad(ticket, taskControlFixture(revision, status), nil) {
		t.Fatal("load dropped")
	}
}
func TestBotTaskControlsRetryReviewAndFailureRetention(t *testing.T) {
	var c TaskControls
	scope := TaskScope{"host-account", "b"}
	loadTaskControl(t, &c, scope, 7, "failed")
	if got := c.Snapshot(scope); len(got.Tasks) != 1 {
		t.Fatal("foreign task leaked")
	}
	c.Select(scope, "t")
	c.Prepare(scope, "retry", "t")
	review := c.Snapshot(scope).Review
	if review == nil || review.Command.RequestID == "" {
		t.Fatal("missing review/id")
	}
	id := review.Command.RequestID
	ticket, command, ok := c.BeginAction(scope)
	if !ok || command.RequestID != id {
		t.Fatal("action not started")
	}
	if _, _, ok = c.BeginAction(scope); ok {
		t.Fatal("duplicate click admitted")
	}
	c.FinishAction(ticket, pluginrpc.DshbotTaskResult{}, errors.New("synthetic timeout"))
	got := c.Snapshot(scope)
	if got.Review == nil || got.Review.Command.RequestID != id || got.Error == "" || got.SelectedID != "t" {
		t.Fatal("failure discarded operation")
	}
	// Refresh advances authority, but must not silently rebase the reviewed write.
	ticket, ok = c.BeginLoad(scope)
	if !ok {
		t.Fatal("refresh denied")
	}
	c.FinishLoad(ticket, taskControlFixture(8, "failed"), nil)
	if _, _, ok = c.BeginAction(scope); ok {
		t.Fatal("stale revision accepted")
	}
	c.Dismiss(scope)
	c.Prepare(scope, "retry", "t")
	if c.Snapshot(scope).Review.Command.RequestID != id {
		t.Fatal("refresh/dismiss lost retry identity")
	}
	ticket, command, ok = c.BeginAction(scope)
	if !ok {
		t.Fatal("reviewed retry denied")
	}
	view := taskControlFixture(9, "failed")
	view.Value.Tasks = append(view.Value.Tasks, pluginrpc.DshbotTask{ID: "new", FromID: "a", ToID: "b", Status: "queued", ParentTaskID: "t", RetryRequestID: id})
	c.FinishAction(ticket, pluginrpc.DshbotTaskResult{View: view, TaskID: "new", OriginalTaskID: "t", WakeError: "synthetic wake unavailable"}, nil)
	got = c.Snapshot(scope)
	if got.Review != nil || got.SelectedID != "new" || got.Revision != 9 || got.Notice == "" || got.Error != "" {
		t.Fatalf("bad settled state: %+v", got)
	}
}
func TestBotTaskControlsFenceHostAccountBotAndLoadFailures(t *testing.T) {
	var c TaskControls
	a := TaskScope{"host-account-a", "b"}
	b := TaskScope{"host-account-b", "b"}
	c.Bind(a)
	old, _ := c.BeginLoad(a)
	c.Bind(b)
	if c.FinishLoad(old, taskControlFixture(7, "queued"), nil) {
		t.Fatal("old account result applied")
	}
	c.Bind(a)
	if c.FinishLoad(old, taskControlFixture(7, "queued"), nil) {
		t.Fatal("navigation back revived old request")
	}
	loadTaskControl(t, &c, a, 7, "queued")
	c.Select(a, "t")
	c.Prepare(a, "pause", "t")
	op, _, _ := c.BeginAction(a)
	c.Bind(TaskScope{"host-account-a", "other-bot"})
	if c.FinishAction(op, pluginrpc.DshbotTaskResult{View: taskControlFixture(8, "paused")}, nil) {
		t.Fatal("old Bot write result applied")
	}
	loadTaskControl(t, &c, a, 8, "queued")
	refresh, _ := c.BeginLoad(a)
	c.FinishLoad(refresh, nil, errors.New("read failed"))
	c.Prepare(a, "stop", "")
	if got := c.Snapshot(a); got.Loaded || got.Review != nil || len(got.Tasks) != 1 {
		t.Fatal("failed refresh enabled writes or lost visible evidence")
	}
}
func TestBotTaskControlsRejectChangedCanonicalSession(t *testing.T) {
	var c TaskControls
	a := TaskScope{"host", "b"}
	loadTaskControl(t, &c, a, 1, "queued")
	c.Prepare(a, "stop", "")
	ticket, _, ok := c.BeginAction(a)
	if !ok {
		t.Fatal("stop denied")
	}
	view := taskControlFixture(2, "queued")
	view.Value.Items[0].SessionID = "replacement-session"
	c.FinishAction(ticket, pluginrpc.DshbotTaskResult{View: view, StopAccepted: true}, nil)
	if got := c.Snapshot(a); got.Error == "" || got.SessionID != "s-b" || got.Review == nil {
		t.Fatal("changed canonical identity accepted")
	}
}
