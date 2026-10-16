package session

// AgentNavigationPane creates only an addressed-history owner. It never
// attaches, selects, follows, promotes, or copies an existing navigation token.
func (s *Store) AgentNavigationPane(id string) *Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agentNavigationPanes == nil {
		s.agentNavigationPanes = map[string]*Store{}
	}
	if pane := s.agentNavigationPanes[id]; pane != nil {
		return pane
	}
	pane := New()
	pane.agentNavigationOwner = s
	pane.agentNavigationPaneID = id
	s.agentNavigationPanes[id] = pane
	return pane
}
func (s *Store) AgentNavigationPaneByID(id string) *Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agentNavigationPanes[id]
}
func (s *Store) ReleaseAgentNavigationPane(id string) {
	s.mu.Lock()
	pane := s.agentNavigationPanes[id]
	delete(s.agentNavigationPanes, id)
	s.mu.Unlock()
	if pane != nil {
		pane.CloseAgentNavigation()
		pane.mu.Lock()
		pane.client = nil
		pane.mux = nil
		pane.planHistoryBinding = ""
		pane.mu.Unlock()
	}
}
func (s *Store) AgentNavigationRootStore() *Store {
	if s.agentNavigationOwner != nil {
		return s.agentNavigationOwner
	}
	return s
}

// UI calls this before rendering. Every read/action also checks the root
// identity live, so a Host/session change rejects old pane actions immediately.
func (s *Store) SyncAgentNavigationPane() {
	owner := s.agentNavigationOwner
	if owner == nil {
		return
	}
	owner.mu.Lock()
	client, mux, root, binding, generation := owner.client, owner.mux, owner.selectedID, owner.planHistoryBinding, owner.followGeneration
	items := append([]Item(nil), owner.items...)
	notify, mode := owner.notify, owner.promptMode
	owner.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != client || s.mux != mux || s.selectedID != root || s.followGeneration != generation || s.planHistoryBinding != binding {
		s.invalidateAgentNavigationLocked()
	}
	s.client, s.mux, s.selectedID, s.planHistoryBinding, s.followGeneration = client, mux, root, binding, generation
	s.items, s.notify, s.promptMode = items, notify, mode
}

// Caller owns this pane's lock. The root never takes a pane lock while holding
// its own lock, including release, so this ownership check has one lock order.
func (s *Store) agentPaneCurrentLocked() bool {
	owner := s.agentNavigationOwner
	if owner == nil {
		return true
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	return owner.agentNavigationPanes[s.agentNavigationPaneID] == s && s.client == owner.client && s.mux == owner.mux && s.selectedID == owner.selectedID && s.planHistoryBinding == owner.planHistoryBinding && s.followGeneration == owner.followGeneration
}
