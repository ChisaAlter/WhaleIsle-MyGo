package settings

// LeaveServedForms discards only the feature-owned plugin forms. Namespace
// browsing and unrelated model drafts keep their existing ownership contract.
// In-flight writes already captured their submission; leaving retires the UI
// drafts without pretending to cancel that Host operation.
func (s *Store) LeaveServedForms() {
	s.mu.Lock()
	for _, ns := range []string{"agent-loop", "subagent", "subagent-model-selection-settings", "web", "web-search-deepseek", "web-deepseek-api", "pwsh-sandbox", "bash-sandbox"} {
		delete(s.drafts, ns)
		delete(s.dirty, ns)
		delete(s.resets, ns)
		delete(s.draftRevisions, ns)
	}
	if s.webSearchKey != nil {
		s.webSearchKey.state.Draft = ""
	}
	s.mu.Unlock()
	s.fire()
}
