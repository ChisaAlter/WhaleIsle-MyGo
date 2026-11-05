package interaction

import "testing"

func TestParityPendingRankAndDiscussOwnership(t *testing.T) {
	s := New()
	request := userQuestionsRequest{Questions: []userQuestionItem{{ID: "p", Question: "Approve?", Detail: "# Plan", Options: []userQuestionOption{{Label: "Yes"}}, Intent: &planReviewIntent{Kind: "plan-review", Approve: "Yes", CallID: "plan-call"}}}}
	s.questionDrafts["plan"] = newDraftSession(request)
	s.pending = []Pending{{EventID: "ordinary", Event: EventUserQuestions}, {EventID: "plan", Event: EventUserQuestions, CallID: "live-plan"}, {EventID: "approval", Event: EventApproval}, {EventID: "new-ordinary", Event: EventUserQuestions}}
	view := s.Snapshot()
	// Installed ui-session selects plan-review (2), questions (1), approval (0),
	// and the latest publication wins within equal precedence.
	if len(view.Pending) != 4 || view.Pending[0].EventID != "plan" || view.Pending[1].EventID != "new-ordinary" || view.Pending[2].EventID != "ordinary" || view.Pending[3].EventID != "approval" {
		t.Fatalf("rank: %+v", view.Pending)
	}
	s.DismissPlanReview("plan")
	view = s.Snapshot()
	card := view.QuestionCards["plan"]
	if !card.Hidden || !card.PlanReviewHidden {
		t.Fatal("Discuss retained composer takeover")
	}
	s.ReopenPlanReview("plan")
	card = s.Snapshot().QuestionCards["plan"]
	if card.Hidden || card.PlanReviewHidden {
		t.Fatal("same review failed to reopen")
	}
}
