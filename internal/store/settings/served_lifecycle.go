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
	s.unlockPublish()
	s.fire()
}

// ClearTransientErrors clears page-level form and immediate preference errors
// when leaving a settings section or closing settings, preventing a failure on
// one page from polluting unrelated pages. Initial describe failures (when the
// current Host has not been described yet) are preserved until explicit retry.
func (s *Store) ClearTransientErrors() {
	s.mu.Lock()
	changed := false
	if s.describedClient == s.client && s.err != nil {
		s.err = nil
		changed = true
	}
	if len(s.preferenceErrors) > 0 {
		s.preferenceErrors = nil
		changed = true
	}
	if changed {
		s.viewStale = true
	}
	s.unlockPublish()
	if changed {
		s.fire()
	}
}
