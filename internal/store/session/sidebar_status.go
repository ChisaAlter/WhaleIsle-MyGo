package session

// These are runtime attention hints, not persisted Session data. Only real Host
// running observations establish transitions; catalog fallbacks never call this.
func (s *Store) observeSidebarRunningLocked(id string, running, beforeBaseline bool) {
	if id == "" {
		return
	}
	if s.sidebarRunning == nil {
		s.sidebarRunning = map[string]bool{}
	}
	if s.completionUnread == nil {
		s.completionUnread = map[string]bool{}
	}
	previous, known := s.sidebarRunning[id]
	s.sidebarRunning[id] = running
	main := s.mainConversationReadID
	if main == "" {
		main = s.selectedID
	}
	if running || id == main {
		delete(s.completionUnread, id)
	} else if previous || !known && beforeBaseline {
		s.completionUnread[id] = true
	}
}
func (s *Store) reconcileSidebarStatusesLocked(items []Item) {
	present := map[string]bool{}
	for _, it := range items {
		present[it.SessionID] = true
		s.observeSidebarRunningLocked(it.SessionID, it.Running, false)
	}
	for id := range s.sidebarRunning {
		if !present[id] {
			delete(s.sidebarRunning, id)
			delete(s.completionUnread, id)
		}
	}
}

// MarkMainConversationViewed is presentation-only: it cannot select or promote
// a session, grant capabilities, or start a reader. Aside panes never call it.
func (s *Store) MarkMainConversationViewed(id string) {
	s.mu.Lock()
	changed := s.completionUnread[id]
	s.mainConversationReadID = id
	delete(s.completionUnread, id)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}
func (s *Store) resetSidebarStatusesLocked() {
	s.sidebarRunning = nil
	s.completionUnread = nil
	s.mainConversationReadID = ""
}

// Copy known Host faces for passive status presentation. This does not create
// question cards, start observers, or turn a status badge into reply authority.
func (s *Store) SidebarQuestionProjections() map[string]QuestionProjectionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]QuestionProjectionState, len(s.questionProjections))
	for id, p := range s.questionProjections {
		p.Questions = append([]byte(nil), p.Questions...)
		p.Inbox = append([]byte(nil), p.Inbox...)
		out[id] = p
	}
	return out
}

// Releasing a child route is not itself proof the parent was viewed: callers
// may be switching to another session. The actual main surface marks its read.
func (s *Store) EndMainConversationChildView() {
	s.mu.Lock()
	s.mainConversationReadID = ""
	s.mu.Unlock()
}
