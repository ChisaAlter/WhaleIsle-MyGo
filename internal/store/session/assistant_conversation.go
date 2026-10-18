package session

// NavigationCurrent fences a delayed assistant ensure/open against both Host
// replacement and navigating away/back while the RPC was in flight.
func (t *ComposerTarget) NavigationCurrent() bool {
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	return t.currentLocked() && t.store.selectedID == t.SessionID && t.store.followGeneration == t.followGeneration
}
