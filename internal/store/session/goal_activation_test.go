package session

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestGoalActivationProjection(t *testing.T) {
	s := New()
	s.applyGoalView(&api.GoalView{Id: "g", Revision: 2, Phase: "active", Activation: "disarmed", BlockedReason: &api.GoalViewBlockedReason{Code: "limit", Message: "Round limit"}})
	g := s.planBoard.Goal
	if g.Phase != "active" || g.Activation != "disarmed" || g.BlockedReason.Message != "Round limit" {
		t.Fatalf("lost Host state: %+v", g)
	}
	clone := clonePlanBoard(s.planBoard)
	clone.Goal.BlockedReason.Message = "changed"
	if g.BlockedReason.Message != "Round limit" {
		t.Fatal("snapshot aliases blocker")
	}
	ApplyPlanBoardEvent(&s.planBoard, json.RawMessage(`{"type":"goal/change","data":{"operation":"edit","goal":{"id":"g","revision":2,"phase":"active","objective":"same"}}}`))
	if s.planBoard.Goal.Activation != "disarmed" {
		t.Fatal("same revision discarded live activation")
	}
	ApplyPlanBoardEvent(&s.planBoard, json.RawMessage(`{"type":"goal/change","data":{"operation":"block","goal":{"id":"g","revision":3,"phase":"blocked","objective":"same","blockedReason":{"code":"limit","message":"Wait"}}}}`))
	if s.planBoard.Goal.Activation != "" || s.planBoard.Goal.BlockedReason.Message != "Wait" {
		t.Fatalf("new revision projection: %+v", s.planBoard.Goal)
	}
	s.applyGoalView(&api.GoalView{Id: "g", Revision: 4, Activation: "armed"})
	if s.planBoard.Goal.Phase != "" {
		t.Fatal("activation must not replace phase")
	}
}

func TestGoalRefreshOnlyAtLifecycleEdges(t *testing.T) {
	for _, raw := range []string{`{"type":"snapshot"}`, `{"type":"event","event":{"type":"goal/change"}}`, `{"type":"event","event":{"type":"turn/start"}}`, `{"type":"event","event":{"type":"turn/end"}}`} {
		if !goalRefreshFollowFrame(json.RawMessage(raw)) {
			t.Fatalf("missing edge %s", raw)
		}
	}
	for _, raw := range []string{`{"type":"assistant-stream"}`, `{"type":"event","event":{"type":"tool/call"}}`, `{`} {
		if goalRefreshFollowFrame(json.RawMessage(raw)) {
			t.Fatalf("unexpected read %s", raw)
		}
	}
}
