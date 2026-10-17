package interaction

import (
	"context"
	"encoding/json"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// SubmissionScope is supplied by an explicitly opened session occurrence. It
// binds one Host and child; validity is checked again for each user gesture.
type SubmissionScope interface {
	Client() *dsh.Client
	Context() context.Context
	AgentID() string
	Valid() bool
}

// ScopedStore shares drafts and projections, but never the selected parent's
// submission authority. All methods which can send are overridden below.
type ScopedStore struct {
	*Store
	scope SubmissionScope
}

func NewScopedStore(s *Store, scope SubmissionScope) *ScopedStore {
	return &ScopedStore{Store: s, scope: scope}
}
func (s *ScopedStore) valid() bool {
	return s != nil && s.Store != nil && s.scope != nil && s.scope.Context() != nil && s.scope.Context().Err() == nil && s.scope.AgentID() != "" && s.scope.Valid()
}
func (s *ScopedStore) pendingLocked(id string) *Pending {
	if id == "" || s.client != s.scope.Client() {
		return nil
	}
	p := s.findPending(id)
	if p == nil || p.AgentID != s.scope.AgentID() {
		return nil
	}
	return p
}
func (s *ScopedStore) allows(id string) bool {
	if !s.valid() {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingLocked(id) != nil
}
func (s *ScopedStore) resolveScoped(id string, out dsh.EventResultOutcome) {
	if !s.valid() {
		return
	}
	s.mu.Lock()
	p := s.pendingLocked(id)
	if p == nil || p.Sending || p.Continued || s.waiters[id] == nil {
		s.mu.Unlock()
		return
	}
	waiter := s.waiters[id]
	out.Delivery = &dsh.EventDelivery{Client: s.scope.Client(), EventID: id, Context: s.scope.Context(), Valid: func() bool {
		if !s.valid() {
			return false
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.pendingLocked(id) != nil && s.waiters[id] == waiter
	}}
	p.Sending, p.Error = true, ""
	s.responses[id] = out
	waiter <- out
	s.mu.Unlock()
	s.fire()
}
func (s *ScopedStore) ResolveAllow(id string) {
	if !s.allows(id) {
		return
	}
	s.mu.Lock()
	p := s.pendingLocked(id)
	approval := p != nil && p.Event == EventApproval
	s.mu.Unlock()
	if approval {
		s.resolveScoped(id, approvalOutcome(outcomeAllowedOnce))
	}
}
func (s *ScopedStore) ResolveAllowAlways(id string) { s.ResolveAllow(id) }
func (s *ScopedStore) ResolveDeny(id string) {
	if !s.allows(id) {
		return
	}
	s.mu.Lock()
	p := s.pendingLocked(id)
	question := p != nil && p.Event == EventUserQuestions
	s.mu.Unlock()
	if question {
		s.resolveScoped(id, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "UserQuestionError", Code: "ASK_CANCELLED", Message: "User declined to answer"}})
		return
	}
	s.resolveScoped(id, approvalOutcome(outcomeRejected))
}
func (s *ScopedStore) sendAnswer(id string, answer userQuestionsAnswer) {
	if !s.allows(id) {
		return
	}
	s.mu.Lock()
	p := s.pendingLocked(id)
	if p == nil || p.Sending {
		s.mu.Unlock()
		return
	}
	continued, agent, call := p.Continued, p.AgentID, p.CallID
	s.mu.Unlock()
	if continued {
		out := api.AskUserQuestionAnswer{Answers: make([]api.AskUserQuestionAnswerAnswersItem, 0, len(answer.Answers))}
		for _, item := range answer.Answers {
			out.Answers = append(out.Answers, api.AskUserQuestionAnswerAnswersItem{Id: item.ID, Selected: item.Selected, Custom: item.Custom})
		}
		s.submitQuestionAnswerBound(s.scope.Context(), agent, call, out, s.scope.Client(), func() bool { return s.allows(id) })
		return
	}
	raw, err := json.Marshal(answer)
	if err == nil {
		s.resolveScoped(id, dsh.EventResultOutcome{Kind: "result", Value: raw})
	}
}
func (s *ScopedStore) ResolveQuestions(id string) {
	if !s.allows(id) {
		return
	}
	s.mu.Lock()
	ds := s.questionDrafts[id]
	if ds == nil || planReviewOf(ds.req.Questions) != nil || !ds.answerValid() {
		s.mu.Unlock()
		return
	}
	answer := ds.buildAnswer()
	s.mu.Unlock()
	s.sendAnswer(id, answer)
}
func (s *ScopedStore) ApprovePlanReview(id string) {
	if !s.allows(id) {
		return
	}
	s.mu.Lock()
	p, ds := s.pendingLocked(id), s.questionDrafts[id]
	if p == nil || p.Event != EventUserQuestions || p.Sending || ds == nil {
		s.mu.Unlock()
		return
	}
	review := planReviewOf(ds.req.Questions)
	s.mu.Unlock()
	if review != nil {
		s.sendAnswer(id, userQuestionsAnswer{Answers: []userQuestionsAnswerItem{{ID: review.ID, Selected: []string{review.Approve.Label}}}})
	}
}
func (s *ScopedStore) DismissPlanReview(id string) {
	if !s.allows(id) {
		return
	}
	s.mu.Lock()
	p, ds := s.pendingLocked(id), s.questionDrafts[id]
	if p == nil || p.Event != EventUserQuestions || p.Sending || ds == nil || planReviewOf(ds.req.Questions) == nil {
		s.mu.Unlock()
		return
	}
	if p.CallID != "" {
		ds.planReviewHidden = true
		s.mu.Unlock()
		s.fire()
		return
	}
	s.mu.Unlock()
	s.resolveScoped(id, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "UserQuestionError", Code: "ASK_CANCELLED", Message: "the user cancelled ask_user_question"}})
}
func (s *ScopedStore) Retry(id string) {
	if !s.allows(id) {
		return
	}
	s.mu.Lock()
	p := s.pendingLocked(id)
	if p == nil || p.Error == "" {
		s.mu.Unlock()
		return
	}
	if p.Continued {
		ds := s.questionDrafts[id]
		plan := ds != nil && planReviewOf(ds.req.Questions) != nil
		s.mu.Unlock()
		if plan {
			s.ApprovePlanReview(id)
		} else {
			s.ResolveQuestions(id)
		}
		return
	}
	out, ok := s.responses[id]
	s.mu.Unlock()
	if ok {
		s.resolveScoped(id, out)
	}
}
func (s *ScopedStore) ContinueQuestion(agent, id, question string) (string, string) {
	if agent != s.scope.AgentID() || !s.allows(id) {
		return "", ""
	}
	return s.continueQuestion(agent, id, question, s.ResolveQuestions)
}
func (s *ScopedStore) SubmitQuestionBatch(agent, id string) (string, string) {
	if agent != s.scope.AgentID() || !s.allows(id) {
		return "", ""
	}
	return s.submitQuestionBatch(agent, id, s.ResolveQuestions)
}
func (s *ScopedStore) SkipQuestionAndContinue(agent, id, question string) (string, string) {
	if agent != s.scope.AgentID() || !s.allows(id) {
		return "", ""
	}
	return s.skipQuestionAndContinue(agent, id, question, s.ResolveQuestions)
}

// A retained surface cannot turn an explicit decision into global auto-allow.
func (*ScopedStore) SetAutoAllow(bool) {}
