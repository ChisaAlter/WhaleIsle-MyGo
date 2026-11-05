package session

// A user gesture may submit only to the exact conversation retained by this
// occurrence. This does not grant catalog actions or shared approvals.
func (s *Store) retainedAgentPromptAllowedLocked(token uint64, address AgentSessionAddress) bool {
	return s.retainedAgentReadValidLocked() && !s.agentRetainedRead.restoredReadOnly && s.agentNavigationOwnedLocked() &&
		s.agentNavigation.view.Token == token && s.agentNavigation.view.Address == address &&
		address == s.agentRetainedRead.address && address.Mode == "continuable" &&
		s.agentNavigation.view.Process.Live && s.agentParentCanPromptLocked(address)
}
func (s *Store) agentPromptActionAdmissionLocked(token uint64, address AgentSessionAddress, explicit bool) bool {
	if explicit && s.agentRetainedRead != nil {
		return s.retainedAgentExplicitGestureAllowedLocked(token, address)
	}
	return s.agentNavigationActionAllowedLocked(token, address)
}

// PromptAgentNavigationExplicit is invoked only by the child editor's send
// gesture. Background and legacy callers retain selected-root authorization.
func (s *Store) PromptAgentNavigationExplicit(token uint64, address AgentSessionAddress, text string) {
	if address.Mode != "continuable" {
		return
	}
	s.agentNavigationAction(token, []AgentSessionAddress{address}, trimPrompt(text), true, true)
}

func (s *Store) PromptAgentNavigationGesture(token uint64, address AgentSessionAddress, text string, accelerated bool, preference ...string) {
	if address.Mode != "continuable" {
		return
	}
	s.mu.Lock()
	images := len(s.agentNavigation.view.PromptImages)
	s.mu.Unlock()
	if accelerated && trimPrompt(text) == "" && images == 0 {
		s.SteerAgentQueue(token, address)
		return
	}
	preferred := "queue"
	if len(preference) > 0 {
		preferred = preference[0]
	}
	s.agentNavigationAction(token, []AgentSessionAddress{address}, trimPrompt(text), true, true, childPromptGesture{accelerated: accelerated, preferred: preferred})
}

// InterruptRetainedAgentExplicit is available only for the already-open aside.
// It cannot address sibling rows or perform a catalog-wide interruption.
func (s *Store) InterruptRetainedAgentExplicit(token uint64, address AgentSessionAddress) {
	s.mu.Lock()
	allowed := s.retainedAgentExplicitGestureAllowedLocked(token, address)
	s.mu.Unlock()
	if !allowed {
		return
	}
	s.agentNavigationAction(token, []AgentSessionAddress{address}, "", false, true)
}
