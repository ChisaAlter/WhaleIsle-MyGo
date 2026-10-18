package interaction

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh"
)

var historyCall = json.RawMessage(`{"type":"tool/call","data":{"callId":"call-1","name":"ask_user_question","arguments":"{\"questions\":[{\"id\":\"q\",\"question\":\"Pick\",\"options\":[{\"label\":\"A\"},{\"label\":\"B\"}]}]}"}}`)
var historySettled = json.RawMessage(`{"active":[],"settled":[{"callId":"call-1","answers":[{"id":"q","selected":["B"],"custom":"detail"}]}]}`)

func TestQuestionHideRetainsWaiterDraftAndReopens(t *testing.T) {
	s := New()
	s.pending = []Pending{{AgentID: "A", EventID: "e", CallID: "call-1", Event: EventUserQuestions}}
	s.waiters["e"] = make(chan dsh.EventResultOutcome, 1)
	s.questionDrafts["e"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{{ID: "q", Question: "Pick"}}})
	s.SetQuestionCustom("e", "q", "typed")
	s.HideQuestion("e")
	if !s.SnapshotForAgent("A").QuestionCards["e"].Hidden || len(s.waiters["e"]) != 0 || *s.QuestionCustomPtr("e", "q") != "typed" {
		t.Fatal("hide sent/dropped answer")
	}
	s.ReopenQuestion("e")
	if s.SnapshotForAgent("A").QuestionCards["e"].Hidden {
		t.Fatal("did not reopen")
	}
	s.OnCancel(dsh.EventsCancel{EventID: "e"})
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("hidden card blocked Host cancel")
	}
}
func TestQuestionHistoryRehydratesAndRejectsStaleProjection(t *testing.T) {
	s := New()
	s.SyncProjectedQuestions("A", historySettled, emptyInboxFixture, 8, 8)
	s.SyncQuestionHistory("A", []json.RawMessage{historyCall}, historySettled, emptyInboxFixture, 8, 8)
	v := s.SnapshotForAgent("A").QuestionHistory
	if len(v) != 1 || v[0].Rows[0].Selected[0] != "B" || v[0].Rows[0].Custom != "detail" {
		t.Fatalf("missing recorded answers: %+v", v)
	}
	if len(s.SnapshotForAgent("B").QuestionHistory) != 0 {
		t.Fatal("cross-session leak")
	}
	s.ReviewQuestionHistory("A", "call-1")
	if !s.SnapshotForAgent("A").QuestionHistory[0].Open {
		t.Fatal("review failed")
	}
	v[0].Rows[0].Selected[0] = "mutated"
	if s.SnapshotForAgent("A").QuestionHistory[0].Rows[0].Selected[0] != "B" {
		t.Fatal("snapshot alias")
	}
	stale := json.RawMessage(`{"active":[],"settled":[{"callId":"call-1","answers":[{"id":"q","selected":["OLD"]}]}]}`)
	s.SyncProjectedQuestions("A", stale, emptyInboxFixture, 2, 2)
	s.SyncQuestionHistory("A", []json.RawMessage{historyCall}, stale, emptyInboxFixture, 2, 2)
	if s.SnapshotForAgent("A").QuestionHistory[0].Rows[0].Selected[0] != "B" {
		t.Fatal("stale history rolled back")
	}
	s.Detach()
	if len(s.SnapshotForAgent("A").QuestionHistory) != 0 {
		t.Fatal("Host replacement kept history")
	}
}
func TestQuestionHistoryRejectsAmbiguousPairsAndLiveWins(t *testing.T) {
	s := New()
	bad := json.RawMessage(`{"active":[],"settled":[{"callId":"call-1","answers":[{"id":"wrong","selected":["B"]}]}]}`)
	s.SyncProjectedQuestions("A", bad, emptyInboxFixture)
	s.SyncQuestionHistory("A", []json.RawMessage{historyCall}, bad, emptyInboxFixture, 0, 0)
	if len(s.SnapshotForAgent("A").QuestionHistory) != 0 {
		t.Fatal("invented answer pairing")
	}
	s.SyncProjectedQuestions("A", historySettled, emptyInboxFixture)
	s.SyncQuestionHistory("A", []json.RawMessage{historyCall}, historySettled, emptyInboxFixture, 0, 0)
	s.pending = []Pending{{AgentID: "A", EventID: "live", CallID: "call-1", Event: EventUserQuestions}}
	s.questionDrafts["live"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{{ID: "q"}}})
	s.HideQuestion("live")
	s.ReviewQuestionHistory("A", "call-1")
	if s.SnapshotForAgent("A").QuestionCards["live"].Hidden || len(s.SnapshotForAgent("A").QuestionHistory) != 0 {
		t.Fatal("stale history replaced live card")
	}
}
func TestQuestionHistoryQueuedAnswersRemainReadOnly(t *testing.T) {
	inbox := json.RawMessage(`{"next-step":[],"next-turn":[{"source":{"kind":"user-question-reply","callId":"call-1"},"content":[{"type":"text","text":"{\"kind\":\"answer_to_pending_question\",\"callId\":\"call-1\",\"answers\":[{\"id\":\"q\",\"selected\":[\"A\"]}]}"}]}]}`)
	s := New()
	s.SyncProjectedQuestions("A", continuedFixture, inbox, 4, 4)
	s.SyncQuestionHistory("A", []json.RawMessage{historyCall}, continuedFixture, inbox, 4, 4)
	v := s.SnapshotForAgent("A")
	if len(v.Pending) != 0 || len(v.QuestionHistory) != 1 || !v.QuestionHistory[0].Queued {
		t.Fatalf("queued answer was editable or lost: %+v", v)
	}
	s.ReviewQuestionHistory("A", "call-1")
	if len(s.SnapshotForAgent("A").Pending) != 0 {
		t.Fatal("review invented submit path")
	}
	s.SyncProjectedQuestions("A", continuedFixture, emptyInboxFixture, 5, 5)
	s.SyncQuestionHistory("A", []json.RawMessage{historyCall}, continuedFixture, emptyInboxFixture, 5, 5)
	v = s.SnapshotForAgent("A")
	if len(v.Pending) != 1 || len(v.QuestionHistory) != 0 {
		t.Fatal("discard did not restore editable draft")
	}
}
