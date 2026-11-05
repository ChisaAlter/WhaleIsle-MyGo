package interaction

// ContinueQuestion implements free-text Enter and the primary Next/Submit
// action. Unlike the pager, it requires an answer on the current page; its final
// page validates the full batch and focuses the first incomplete question.
func (s *Store) ContinueQuestion(agentID, eventID, questionID string) (focusID, feedback string) {
	return s.continueQuestion(agentID, eventID, questionID, s.ResolveQuestions)
}

func (s *Store) continueQuestion(agentID, eventID, questionID string, submit func(string)) (focusID, feedback string) {
	defer s.persistQuestionDrafts()
	s.mu.Lock()
	ds := s.questionWizardLocked(agentID, eventID)
	if ds == nil || ds.index < 0 || ds.index >= len(ds.req.Questions) || ds.req.Questions[ds.index].ID != questionID {
		s.mu.Unlock()
		return "", ""
	}
	if !questionAnswered(ds.answers[questionID]) {
		s.mu.Unlock()
		return questionID, "unanswered"
	}
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
