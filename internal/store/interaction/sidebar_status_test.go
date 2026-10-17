package interaction

import (
	"encoding/json"
	"testing"
)

func TestSidebarPendingProjection(t *testing.T) {
	q := json.RawMessage(`{"active":[{"callId":"q","state":"continued","questions":[{"id":"q","question":"Pick"}]}],"settled":[]}`)
	empty := json.RawMessage(`{"next-step":[],"next-turn":[]}`)
	if ProjectedPendingStatus(q, nil) != "" {
		t.Fatal("missing inbox invented continued pending")
	}
	if ProjectedPendingStatus(q, empty) != "question" {
		t.Fatal("known unanswered question omitted")
	}
	queued := json.RawMessage(`{"next-step":[{"source":{"kind":"user-question-reply","callId":"q"}}],"next-turn":[]}`)
	if ProjectedPendingStatus(q, queued) != "" {
		t.Fatal("queued answer still needs response")
	}
	plan := json.RawMessage(`{"active":[{"callId":"plan","state":"open","questions":[{"id":"p","question":"Review","detail":"Plan body","intent":{"kind":"plan-review","approve":"Yes","callId":"doc"},"options":[{"label":"Yes"},{"label":"No"}]}]}],"settled":[]}`)
	if ProjectedPendingStatus(plan, nil) != "plan-review" {
		t.Fatal("explicit plan intent lost")
	}
	s := New()
	s.questionAgentLocked("a").answered["q"] = true
	if s.ProjectedSidebarStatus("a", q, empty) != "" {
		t.Fatal("delivered answer returned as stale badge")
	}
	if s.ProjectedSidebarStatus("b", q, empty) != "question" {
		t.Fatal("answer state leaked across agents")
	}
}
func TestSidebarLivePendingPriority(t *testing.T) {
	s := New()
	s.pending = []Pending{{EventID: "approval", AgentID: "a", Event: EventApproval}, {EventID: "question", AgentID: "a", Event: EventUserQuestions}, {EventID: "done", AgentID: "b", Event: EventApproval, approvalResolved: true}}
	if got := s.PendingStatuses(); got["a"] != "question" || got["b"] != "" {
		t.Fatal(got)
	}
	if PendingStatusRank("plan-review") <= PendingStatusRank("question") || PendingStatusRank("question") <= PendingStatusRank("approval") {
		t.Fatal("original precedence not preserved")
	}
}
