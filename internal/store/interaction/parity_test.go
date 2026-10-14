package interaction

import "testing"

func TestQuestionCustomOptionParity(t *testing.T) {
	s := New()
	s.questionDrafts["e"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{{ID: "q", Options: []userQuestionOption{{Label: "A"}}}}})
	s.SetQuestionCustom("e", "q", "other")
	got := s.questionDrafts["e"].buildAnswer().Answers[0]
	if got.Custom == nil || *got.Custom != "other" || len(got.Selected) != 0 {
		t.Fatalf("custom single option answer: %#v", got)
	}
}
func TestStaleQuestionCannotTargetAnotherCard(t *testing.T) {
	s := New()
	s.pending = []Pending{{EventID: "live", Event: EventUserQuestions}}
	if s.findPending("stale") != nil {
		t.Fatal("stale ID fell back to unrelated card")
	}
}
