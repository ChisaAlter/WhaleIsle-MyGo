package session

// AgentConversationRoutes returns only already verified directory edges. The
// resulting buttons still revalidate via AgentChatTarget when activated.
func (s *Store) AgentConversationRoutes(token uint64, target AgentSessionAddress) (ancestors, siblings []AgentNavigationRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token {
		return nil, nil
	}
	views := append(append([]AgentNavigationView(nil), n.parents...), n.view)
	if n.view.Address != target {
		for _, node := range s.agentCatalogTreeLocked() {
			if node.Row.Address == target {
				if parents, ok := s.agentCatalogPathLocked(node.Path); ok {
					views = append(append([]AgentNavigationView(nil), n.parents...), parents...)
				}
				break
			}
		}
	}
	for _, view := range views {
		s.refreshAgentNavigationRowsLocked(&view)
		if view.Address.validChild() && view.Address != target {
			ancestors = append(ancestors, AgentNavigationRow{Address: view.Address, Label: view.Title})
		}
		if view.SessionID == target.ParentSessionID {
			for _, row := range view.Rows {
				if row.CanOpen() && row.Address.ParentSessionID == target.ParentSessionID {
					siblings = append(siblings, row)
				}
			}
		}
	}
	return ancestors, siblings
}
