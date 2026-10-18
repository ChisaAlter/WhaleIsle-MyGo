package session

// Used only by explicit child-editor operations and their availability view.
// It never upgrades the lease or enables legacy/background submission paths.
func (s *Store) retainedAgentExplicitGestureAllowedLocked(token uint64, address AgentSessionAddress) bool {
	lease := s.agentRetainedRead
	if lease == nil || !lease.restoredReadOnly {
		return s.retainedAgentPromptAllowedLocked(token, address)
	}
	if !lease.restoreVerified {
		return false
	}
	a := &AgentInteractionScope{pane: s, lease: lease, token: token, address: address, explicitGesture: true}
	return s.agentInteractionScopeValidLocked(a)
}
