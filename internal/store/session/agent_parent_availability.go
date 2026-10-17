package session

// Caller holds this pane's mutex. Read the owner's current Host summary rather
// than a copied pane frame: availability may change after the UI was painted.
func (s *Store) agentParentAvailabilityLocked(address AgentSessionAddress) (bool, bool) {
	if !address.validChild() {
		return false, false
	}
	items := s.items
	if owner := s.agentNavigationOwner; owner != nil {
		owner.mu.Lock()
		defer owner.mu.Unlock()
		items = owner.items
	}
	for _, it := range items {
		if it.SessionID == address.ParentSessionID {
			return true, it.AgentAvailable
		}
	}
	return false, false
}
func (s *Store) agentParentCanPromptLocked(address AgentSessionAddress) bool {
	known, available := s.agentParentAvailabilityLocked(address)
	return address.Mode == "continuable" && known && available
}
func (s *Store) agentParentViewLocked(v *AgentNavigationView) {
	v.RetainedPromptAllowed = s.retainedAgentExplicitGestureAllowedLocked(v.Token, v.Address)
	v.RetainedReadOnly = s.agentRetainedRead != nil && !s.agentPaneCurrentLocked()
	v.ParentAvailabilityKnown, v.ParentAvailable = s.agentParentAvailabilityLocked(v.Address)
}
