package session

import (
	"encoding/json"
	"testing"
)

func TestQuestionProjectionSnapshotIncrementAndSelection(t *testing.T) {
	s := New()
	s.selectedID = "A"
	s.applyQuestionValuesLocked(json.RawMessage(`{"userQuestions":{"active":[{"callId":"A-q","state":"continued","questions":[]}],"settled":[]},"inbox":{"next-step":[],"next-turn":[]}}`), 10)
	first := s.questionProjectionLocked()
	if len(first.Questions) == 0 || len(first.Inbox) == 0 || first.QuestionsSeq != 10 || first.InboxSeq != 10 {
		t.Fatalf("full snapshot: %+v", first)
	}
	s.applyQuestionControlLocked(json.RawMessage(`{"type":"projection","sessionId":"A","key":"userQuestions","seq":12,"value":{"active":[],"settled":[{"callId":"A-q"}]}}`))
	s.applyQuestionValuesLocked(json.RawMessage(`{"userQuestions":{"active":[{"callId":"stale"}],"settled":[]},"inbox":{"next-step":[],"next-turn":[]}}`), 11)
	got := s.questionProjectionLocked()
	if got.QuestionsSeq != 12 || got.InboxSeq != 11 {
		t.Fatalf("independent face ordering: %+v", got)
	}
	s.selectedID = "B"
	if got := s.questionProjectionLocked(); got.Questions != nil || got.Inbox != nil {
		t.Fatal("new selection inherited another question projection")
	}
	s.applyQuestionControlLocked(json.RawMessage(`{"type":"projection","sessionId":"A","key":"inbox","seq":13,"value":{"next-step":[],"next-turn":[{"source":{"kind":"user-question-reply","callId":"A-q"}}]}}`))
	if s.questionProjectionLocked().Inbox != nil {
		t.Fatal("A increment mutated B snapshot")
	}
	s.selectedID = "A"
	if got := s.questionProjectionLocked(); got.InboxSeq != 13 || got.QuestionsSeq != 12 {
		t.Fatal("switching back lost authoritative faces")
	}
}
func TestQuestionProjectionControlReconnectBaselineAndNull(t *testing.T) {
	s := New()
	s.selectedID = "A"
	s.applyQuestionControlLocked(json.RawMessage(`{"type":"baseline","value":{"projections":{"A":{"asOfSeq":40,"values":{"userQuestions":{"active":[],"settled":[]},"inbox":{"next-step":[],"next-turn":[]}}},"B":{"asOfSeq":7,"values":{"userQuestions":{"active":[{"callId":"B-q"}],"settled":[]},"inbox":{"next-step":[],"next-turn":[]}}}}}}`))
	if got := s.questionProjectionLocked(); got.QuestionsSeq != 40 || got.InboxSeq != 40 {
		t.Fatalf("A reconnect baseline: %+v", got)
	}
	s.selectedID = "B"
	if got := s.questionProjectionLocked(); got.QuestionsSeq != 7 || got.InboxSeq != 7 {
		t.Fatalf("B reconnect baseline: %+v", got)
	}
	s.applyQuestionValuesLocked(json.RawMessage(`{"userQuestions":null,"inbox":null}`), 8)
	got := s.questionProjectionLocked()
	if string(got.Questions) != `{"active":[],"settled":[]}` || string(got.Inbox) != `{"next-step":[],"next-turn":[]}` {
		t.Fatalf("null did not clear authoritative state: %+v", got)
	}
	got.Questions[0] = '!'
	if s.questionProjectionLocked().Questions[0] == '!' {
		t.Fatal("UI snapshot mutated store projection")
	}
}
func TestQuestionProjectionMalformedFramesDoNotEraseLoadedFaces(t *testing.T) {
	s := New()
	s.selectedID = "A"
	s.applyQuestionValuesLocked(json.RawMessage(`{"userQuestions":{"active":[],"settled":[]},"inbox":{"next-turn":[]}}`), 1)
	s.applyQuestionControlLocked(json.RawMessage(`{"type":"projection","sessionId":"A","key":"other","seq":9,"value":{}}`))
	s.applyQuestionValuesLocked(json.RawMessage(`broken`), 10)
	if got := s.questionProjectionLocked(); got.QuestionsSeq != 1 || got.InboxSeq != 1 {
		t.Fatal("malformed or unrelated update erased face")
	}
}
