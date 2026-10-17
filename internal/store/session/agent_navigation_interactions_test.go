package session

import (
	"encoding/json"
	"fmt"
	"testing"
	"whaleisle/internal/dsh"
)

func TestContinuableChildInteractionScope(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	mux := &dsh.Mux{}
	s.client, s.mux, s.selectedID = client, mux, "root"
	s.planHistoryBinding = fmt.Sprintf("%p\x00root", client)
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
	s.agentNavigation = agentNavigationState{binding: s.planHistoryBinding, root: "root", client: client, mux: mux, generation: s.followGeneration, view: AgentNavigationView{Token: 9, SessionID: "child", Address: address}}
	s.agentNavigation.view.Process.Live = true
	s.applyQuestionValuesForSessionLocked("child", json.RawMessage(`{"userQuestions":{"active":[],"settled":[]},"inbox":{"next-turn":[]}}`), 12)
	got, q, _, ok := s.AgentInteractionTarget(9, address)
	if !ok || got != client || q.QuestionsSeq != 12 || q.Inbox == nil {
		t.Fatal("missing child scope/projections")
	}
	if _, _, _, ok = s.AgentInteractionTarget(8, address); ok {
		t.Fatal("stale token admitted")
	}
	for _, mode := range []string{"one-shot", "external", "unknown"} {
		other := address
		other.Mode = mode
		s.agentNavigation.view.Address = other
		if _, _, _, ok = s.AgentInteractionTarget(9, other); ok {
			t.Fatal("read-only mode admitted", mode)
		}
	}
	s.agentNavigation.view.Address = address
	s.agentNavigation.view.Process.Live = false
	if _, _, _, ok = s.AgentInteractionTarget(9, address); ok {
		t.Fatal("history/paused pane admitted")
	}
	s.agentNavigation.view.Process.Live = true
	s.selectedID = "other"
	if _, _, _, ok = s.AgentInteractionTarget(9, address); ok {
		t.Fatal("switched root admitted")
	}
}
