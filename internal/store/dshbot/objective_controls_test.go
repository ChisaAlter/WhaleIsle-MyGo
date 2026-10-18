package dshbot

import (
	"errors"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func objectiveFixture() pluginrpc.DshbotObjectiveStatus {
	return pluginrpc.DshbotObjectiveStatus{Bot: pluginrpc.DshbotObjectiveBot{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s", Name: "Fixture Bot"}, Objective: "Fixture objective", ObjectiveMaxRounds: 16}, Revision: 4, GoalKnown: true, Goal: &pluginrpc.DshbotObjectiveGoal{Phase: "active", Activation: "armed", RoundsStarted: 2, MaxGoalRounds: 16, Blocker: &pluginrpc.DshbotObjectiveBlocker{Code: "fixture", Message: "unchanged"}}}
}
func objectiveLoaded(t *testing.T) (*ObjectiveControls, RoutineScope) {
	t.Helper()
	c := &ObjectiveControls{}
	k := RoutineScope{HostKey: "fixture-auth", BotID: "b"}
	c.Bind(k)
	ticket, _ := c.BeginLoad(k)
	if !c.FinishLoad(ticket, objectiveFixture(), nil) {
		t.Fatal("fixture load")
	}
	return c, k
}
func TestBotObjectiveReviewScopeAndUnknownOutcome(t *testing.T) {
	c, k := objectiveLoaded(t)
	oldView := c.Snapshot(k).Generation
	c.Prepare(k, "start")
	r := c.Snapshot(k).Review
	if c.ViewCurrent(k, oldView) || r == nil {
		t.Fatal("review identity missing")
	}
	ticket, cmd, ok := c.BeginAction(k, r)
	if !ok || cmd.Action != "start" || c.Snapshot(k).Loaded {
		t.Fatal("explicit action failed")
	}
	if _, _, ok := c.BeginAction(k, r); ok {
		t.Fatal("double start")
	}
	c.FinishAction(ticket, pluginrpc.DshbotObjectiveResult{Attempted: true}, errors.New("synthetic unknown"))
	if c.Snapshot(k).Loaded || c.Current(ticket) {
		t.Fatal("unknown result enabled retry")
	}
	c.Prepare(k, "start")
	if c.Snapshot(k).Review != nil {
		t.Fatal("unknown result permitted review before read")
	}
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, objectiveFixture(), nil)
	c.Prepare(k, "pause")
	if _, _, ok := c.BeginAction(k, r); ok {
		t.Fatal("old confirmation consumed new action")
	}
	current := c.Snapshot(k).Review
	ticket, _, _ = c.BeginAction(k, current)
	c.Bind(RoutineScope{})
	c.Bind(k)
	if c.Current(ticket) || c.FinishAction(ticket, pluginrpc.DshbotObjectiveResult{Status: objectiveFixture()}, nil) {
		t.Fatal("closed/reopened action survived")
	}
}
func TestBotObjectiveSnapshotsAndClearWarnings(t *testing.T) {
	c, k := objectiveLoaded(t)
	c.Prepare(k, "clear")
	snapshot := c.Snapshot(k)
	snapshot.Status.Goal.Blocker.Message = "mutated"
	snapshot.Review.Command.Expected.Goal.Blocker.Message = "mutated"
	if c.Snapshot(k).Status.Goal.Blocker.Message != "unchanged" || c.Snapshot(k).Review.Command.Expected.Goal.Blocker.Message != "unchanged" {
		t.Fatal("goal snapshot aliases")
	}
	r := c.Snapshot(k).Review
	ticket, _, ok := c.BeginAction(k, r)
	if !ok {
		t.Fatal("clear missing")
	}
	result := objectiveFixture()
	result.Bot.Objective = ""
	result.Goal = nil
	result.GoalKnown = false
	c.FinishAction(ticket, pluginrpc.DshbotObjectiveResult{Status: result, Warnings: []string{"fixture live goal clear failed"}}, nil)
	s := c.Snapshot(k)
	if s.Loaded || s.Status.GoalKnown || s.Status.Bot.Objective != "" || len(s.Warnings) != 1 {
		t.Fatal("clear receipt invented live stop or hid warning")
	}
	load, _ := c.BeginLoad(k)
	actual := result
	actual.GoalKnown = true
	actual.Goal = &pluginrpc.DshbotObjectiveGoal{Phase: "active", Activation: "armed", RoundsStarted: 3, MaxGoalRounds: 16}
	c.FinishLoad(load, actual, nil)
	if !c.Snapshot(k).Loaded || c.Snapshot(k).Status.Goal.Phase != "active" || len(c.Snapshot(k).Warnings) != 1 {
		t.Fatal("readback did not preserve real live state/warning")
	}
}
func TestBotObjectiveParentLifecycleIsolation(t *testing.T) {
	var parent RoutineControls
	k := RoutineScope{HostKey: "host-a", BotID: "b"}
	parent.Bind(k)
	c := &parent.Objective
	ticket, _ := c.BeginLoad(k)
	parent.Bind(RoutineScope{HostKey: "host-b", BotID: "b"})
	if c.FinishLoad(ticket, objectiveFixture(), nil) || c.Snapshot(k).Loaded {
		t.Fatal("foreign Host late read")
	}
	parent.Bind(k)
	ticket, _ = c.BeginLoad(k)
	wrong := objectiveFixture()
	wrong.Bot.ID = "other"
	c.FinishLoad(ticket, wrong, nil)
	if c.Snapshot(k).Loaded {
		t.Fatal("wrong Bot load accepted")
	}
}
