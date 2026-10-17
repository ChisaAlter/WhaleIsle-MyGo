package interaction

import "strings"

// ContinueQuestion implements the free-text Enter gesture. The current answer
// must be nonempty; intermediate questions move focus, and the final one either
// points to the first incomplete answer or submits the existing request route.
// Returned feedback is editor feedback, never a transport retry error.
func (s *Store) ContinueQuestion(agentID, eventID, questionID string) (focusID, feedback string) {
	s.mu.Lock()
	p, ds := s.findPending(eventID), s.questionDrafts[eventID]
	if p == nil || p.AgentID != agentID || p.Event != EventUserQuestions || p.Sending || p.Error != "" || ds == nil || ds.hidden || planReviewOf(ds.req.Questions) != nil {
		s.mu.Unlock()
		return "", ""
	}
	index := -1
	for i, q := range ds.req.Questions {
		if q.ID == questionID {
			index = i
			break
		}
	}
	if index < 0 {
		s.mu.Unlock()
		return "", ""
	}
	answer := ds.answers[questionID]
	if answer == nil || (len(answer.selected) == 0 && strings.TrimSpace(answer.custom) == "") {
		s.mu.Unlock()
		return questionID, "unanswered"
	}
	if index+1 < len(ds.req.Questions) {
		next := ds.req.Questions[index+1].ID
		s.mu.Unlock()
		return next, ""
	}
	for _, q := range ds.req.Questions {
		a := ds.answers[q.ID]
		if a == nil || (!a.skipped && len(a.selected) == 0 && strings.TrimSpace(a.custom) == "") {
			s.mu.Unlock()
			return q.ID, "incomplete"
		}
	}
	s.mu.Unlock()
	s.ResolveQuestions(eventID)
	return "", ""
}
