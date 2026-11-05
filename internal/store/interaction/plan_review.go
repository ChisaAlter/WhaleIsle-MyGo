package interaction

import (
	"encoding/json"

	"whaleisle/internal/dsh"
)

// planReviewIntent comes from AskUserQuestionItem.intent, not a guessed tool name.
type planReviewIntent struct {
	Kind    string `json:"kind"`
	Approve string `json:"approve"`
	CallID  string `json:"callId"`
}

type PlanReviewOption struct {
	Label       string
	Description string
}

// PlanReview is the exact question and document an explicit approval answers.
// CallID names the plan document; Pending.CallID names the question RPC request.
type PlanReview struct {
	ID       string
	Question string
	Plan     string
	CallID   string
	Approve  PlanReviewOption
	Decline  *PlanReviewOption
}

type planReviewOpenKey struct {
	client   *dsh.Client
	agentID  string
	identity string
}

// ConsumePlanReviewOpen is the Session-owned once marker for the first visible
// review. Live-to-continued delivery and surface remounts share the same marker;
// hiding a review does not consume it, and manual preview remains independent.
func (s *Store) ConsumePlanReviewOpen(agentID, eventID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, draft := s.findPending(eventID), s.questionDrafts[eventID]
	if agentID == "" || p == nil || p.AgentID != agentID || p.Event != EventUserQuestions || draft == nil || draft.hidden || draft.planReviewHidden {
		return false
	}
	review := planReviewOf(draft.req.Questions)
	if review == nil {
		return false
	}
	identity := "review:" + eventID
	if p.CallID != "" {
		identity = "review:" + p.CallID
	}
	if review.CallID != "" {
		identity = "call:" + review.CallID
	}
	key := planReviewOpenKey{s.client, agentID, identity}
	if s.planReviewOpened[key] {
		return false
	}
	if s.planReviewOpened == nil {
		s.planReviewOpened = make(map[planReviewOpenKey]bool)
	}
	s.planReviewOpened[key] = true
	return true
}

func planReviewOf(questions []userQuestionItem) *PlanReview {
	if len(questions) != 1 {
		return nil
	}
	q := questions[0]
	if q.Intent == nil || q.Intent.Kind != "plan-review" || q.Detail == "" || q.MultiSelect || len(q.Options) > 2 {
		return nil
	}
	var approve *PlanReviewOption
	var decline *PlanReviewOption
	for _, option := range q.Options {
		value := PlanReviewOption{Label: option.Label, Description: option.Description}
		if option.Label == q.Intent.Approve {
			if approve == nil {
				approve = &value
			}
		} else {
			decline = &value
		}
	}
	if approve == nil {
		return nil
	}
	return &PlanReview{ID: q.ID, Question: q.Question, Plan: q.Detail, CallID: q.Intent.CallID, Approve: *approve, Decline: decline}
}

// ApprovePlanReview is the only plan-review submit action. It constructs the
// asker's exact approve label only when the user clicks the dedicated button.
// Generic Continue is deliberately disabled for this presentation kind.
func (s *Store) ApprovePlanReview(eventID string) {
	s.mu.Lock()
	p, draft := s.findPending(eventID), s.questionDrafts[eventID]
	if p == nil || p.Event != EventUserQuestions || p.Sending || draft == nil {
		s.mu.Unlock()
		return
	}
	review := planReviewOf(draft.req.Questions)
	if review == nil {
		s.mu.Unlock()
		return
	}
	continued := p.Continued
	s.mu.Unlock()
	answer := userQuestionsAnswer{Answers: []userQuestionsAnswerItem{{ID: review.ID, Selected: []string{review.Approve.Label}}}}
	if continued {
		s.resolveContinuedQuestions(eventID, answer)
		return
	}
	raw, err := json.Marshal(answer)
	if err != nil {
		return
	} // all fields are strings; retained for wire safety
	s.resolve(eventID, dsh.EventResultOutcome{Kind: "result", Value: raw})
}

// DismissPlanReview returns to discussion. A named request only hides its panel:
// its live waiter/countdown and late-answer recovery remain intact and reachable.
// Without a Host request identity, dismiss must cancel through the waterfall.
func (s *Store) DismissPlanReview(eventID string) {
	s.mu.Lock()
	p, draft := s.findPending(eventID), s.questionDrafts[eventID]
	if p == nil || p.Event != EventUserQuestions || p.Sending || draft == nil || planReviewOf(draft.req.Questions) == nil {
		s.mu.Unlock()
		return
	}
	if p.CallID != "" {
		draft.planReviewHidden = true
		draft.hidden = true
		s.mu.Unlock()
		s.fire()
		return
	}
	s.mu.Unlock()
	s.resolve(eventID, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "UserQuestionError", Code: "ASK_CANCELLED", Message: "the user cancelled ask_user_question"}})
}

// ReopenPlanReview restores the same review without answering it or resetting
// the Host's request timer. A recovered draft shares this state across sources.
func (s *Store) ReopenPlanReview(eventID string) {
	s.mu.Lock()
	if draft := s.questionDrafts[eventID]; draft != nil {
		if draft.hidden || draft.planReviewHidden {
			s.raisePendingLocked(eventID)
		}
		draft.planReviewHidden = false
		draft.hidden = false
	}
	s.mu.Unlock()
	s.fire()
}
