package interaction

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/dsh"
)

func TestQuestionSendingFreezesDraftMutations(t *testing.T) {
	for _, kind := range []string{"answer", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			s := New()
			s.pending = []Pending{{AgentID: "A", EventID: "e", Event: EventUserQuestions}}
			s.waiters["e"] = make(chan dsh.EventResultOutcome, 1)
			ds := newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{
				{ID: "text"}, {ID: "single", Options: []userQuestionOption{{Label: "A"}, {Label: "B"}}},
				{ID: "multi", MultiSelect: true, Options: []userQuestionOption{{Label: "A"}, {Label: "B"}}},
			}})
			s.questionDrafts["e"] = ds
			s.SetQuestionCustom("e", "text", "before\nanswer")
			s.SetQuestionSingleSelect("e", "single", "A")
			s.SetQuestionOptionChecked("e", "multi", "A", true)
			before, _ := json.Marshal(ds.buildAnswer())
			if kind == "answer" {
				s.ResolveQuestions("e")
			} else {
				s.ResolveDeny("e")
			}
			s.SetQuestionCustom("e", "text", "after")
			s.SetQuestionSingleSelect("e", "single", "B")
			s.SetQuestionOptionChecked("e", "multi", "B", true)
			s.SetQuestionSkipped("e", "text", true)
			after, _ := json.Marshal(ds.buildAnswer())
			if string(after) != string(before) {
				t.Fatalf("in-flight %s changed draft: %s -> %s", kind, before, after)
			}
		})
	}
}

func TestQuestionContinueValidatesCurrentAndWholeRequest(t *testing.T) {
	s := New()
	s.pending = []Pending{{AgentID: "A", EventID: "e", Event: EventUserQuestions}}
	s.waiters["e"] = make(chan dsh.EventResultOutcome, 1)
	s.questionDrafts["e"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{{ID: "first"}, {ID: "last"}}})
	if next, status := s.ContinueQuestion("A", "e", "first"); next != "first" || status != "unanswered" {
		t.Fatalf("empty answer continued: %q %q", next, status)
	}
	s.SetQuestionCustom("e", "first", "first\nanswer")
	if next, status := s.ContinueQuestion("A", "e", "first"); next != "last" || status != "" {
		t.Fatalf("did not advance: %q %q", next, status)
	}
	if len(s.waiters["e"]) != 0 {
		t.Fatal("intermediate answer submitted")
	}
	s.SetQuestionCustom("e", "first", "")
	s.SetQuestionCustom("e", "last", "final answer")
	if next, status := s.ContinueQuestion("A", "e", "last"); next != "first" || status != "incomplete" {
		t.Fatalf("missing earlier answer not found: %q %q", next, status)
	}
	s.SetQuestionSkipped("e", "first", true)
	s.NavigateQuestion("A", "e", 1) // Resume the final page after missing-answer redirection.
	if next, _ := s.ContinueQuestion("B", "e", "last"); next != "" || len(s.waiters["e"]) != 0 {
		t.Fatal("foreign session submitted")
	}
	s.ContinueQuestion("A", "e", "last")
	if len(s.waiters["e"]) != 1 || !s.pending[0].Sending {
		t.Fatal("complete final answer not submitted")
	}
	s.ContinueQuestion("A", "e", "last")
	if len(s.waiters["e"]) != 1 {
		t.Fatal("in-flight answer duplicated")
	}
}
