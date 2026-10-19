package interaction

import "strings"

// questionWizardLocked identifies an editable, visible generic request. Page
// navigation never borrows another session and never touches plan review.
func (s *Store) questionWizardLocked(agentID, eventID string) *draftSession {
	p, ds := s.findPending(eventID), s.questionDrafts[eventID]
	if p == nil || p.AgentID != agentID || p.Event != EventUserQuestions || p.Sending || ds == nil || ds.hidden || planReviewOf(ds.req.Questions) != nil {
		return nil
	}
	return ds
}

// NavigateQuestion is the unrestricted pager: answers need not be complete.
// Index lives with the draft, so hide/reopen and live-to-continued retain it.
func (s *Store) NavigateQuestion(agentID, eventID string, index int) bool {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	ds := s.questionWizardLocked(agentID, eventID)
	if ds == nil || index < 0 || index >= len(ds.req.Questions) {
		s.mu.Unlock()
		return false
	}
	ds.index = index
	s.mu.Unlock()
	s.fire()
	return true
}

// AdvanceQuestionChoice follows an explicit single-choice selection, never a
// multiselect toggle or untouched recommended default. Final choices do not send.
func (s *Store) AdvanceQuestionChoice(agentID, eventID, questionID string) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	ds := s.questionWizardLocked(agentID, eventID)
	if ds != nil && ds.index >= 0 && ds.index < len(ds.req.Questions) {
		q := ds.req.Questions[ds.index]
		if q.ID == questionID && !q.MultiSelect && ds.answers[q.ID] != nil && len(ds.answers[q.ID].selected) > 0 && ds.index+1 < len(ds.req.Questions) {
			ds.index++
		}
	}
	s.mu.Unlock()
	s.fire()
}

func questionAnswered(a *answerDraft) bool {
	return a != nil && (len(a.selected) > 0 || strings.TrimSpace(a.custom) != "")
}

// questionSubmitTargetLocked changes only page position. No answer is sent
// until all rows are answered or explicitly skipped.
func questionSubmitTargetLocked(ds *draftSession) (focusID, feedback string) {
	for i, q := range ds.req.Questions {
		a := ds.answers[q.ID]
		if a == nil || (!a.skipped && !questionAnswered(a)) {
			ds.index = i
			return q.ID, "incomplete"
		}
	}
	return "", ""
}

// SubmitQuestionBatch is the original option-button Enter gesture. It validates
// the whole retained batch without selecting the focused option first.
func (s *Store) SubmitQuestionBatch(agentID, eventID string) (focusID, feedback string) {
	return s.submitQuestionBatch(agentID, eventID, s.ResolveQuestions)
}

func (s *Store) submitQuestionBatch(agentID, eventID string, submit func(string)) (focusID, feedback string) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	ds := s.questionWizardLocked(agentID, eventID)
	p := s.findPending(eventID)
	if ds == nil || len(ds.req.Questions) == 0 || p.Error != "" {
		s.mu.Unlock()
		return "", ""
	}
	focusID, feedback = questionSubmitTargetLocked(ds)
	s.mu.Unlock()
	if feedback == "" {
		submit(eventID)
	} else {
		s.fire()
	}
	return
}

// SkipQuestionAndContinue clears exactly the visible answer, advances, and on
// the final question directly validates/submits, without an extra review step.
func (s *Store) SkipQuestionAndContinue(agentID, eventID, questionID string) (focusID, feedback string) {
	return s.skipQuestionAndContinue(agentID, eventID, questionID, s.ResolveQuestions)
}

func (s *Store) skipQuestionAndContinue(agentID, eventID, questionID string, submit func(string)) (focusID, feedback string) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	ds := s.questionWizardLocked(agentID, eventID)
	p := s.findPending(eventID)
	if ds == nil || p.Error != "" || ds.index < 0 || ds.index >= len(ds.req.Questions) || ds.req.Questions[ds.index].ID != questionID || ds.answers[questionID] == nil {
		s.mu.Unlock()
		return "", ""
	}
	s.setQuestionSkippedDraftLocked(eventID, ds.answers[questionID], true)
	if ds.index+1 < len(ds.req.Questions) {
		ds.index++
		focusID = ds.req.Questions[ds.index].ID
		s.mu.Unlock()
		s.fire()
		return
	}
	focusID, feedback = questionSubmitTargetLocked(ds)
	s.mu.Unlock()
	if feedback == "" {
		submit(eventID)
	} else {
		s.fire()
	}
	return
}
