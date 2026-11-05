package session

// ID-only routing facts from the currently verified navigation chain. The
// persisted path grants nothing; every edge must be queried again at restore.
func (s *Store) AgentChatRestoreLineage(token uint64, target AgentSessionAddress) []AgentSessionAddress {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &s.agentNavigation
	if !s.agentPaneCurrentLocked() || !s.agentNavigationOwnedLocked() || n.view.Token != token {
		return nil
	}
	views := append(append([]AgentNavigationView(nil), n.parents...), n.view)
	if len(views) == 0 || views[0].SessionID != s.selectedID {
		return nil
	}
	path := []AgentSessionAddress{}
	// A header can choose an ancestor or sibling from any already-owned parent
	// directory. Reconstruct only the prefix which actually reaches that edge.
	prefix := []AgentSessionAddress{}
	for index, view := range views {
		s.refreshAgentNavigationRowsLocked(&view)
		for _, row := range view.Rows {
			if row.CanOpen() && row.Address == target {
				candidate := append(append([]AgentSessionAddress(nil), prefix...), target)
				if validAgentRestoreLineage(s.selectedID, target, candidate) {
					return candidate
				}
			}
		}
		if index+1 < len(views) {
			next := views[index+1].Address
			found := false
			for _, row := range view.Rows {
				found = found || row.CanOpen() && row.Address == next
			}
			if !found {
				break
			}
			prefix = append(prefix, next)
		}
	}
	for i := 1; i < len(views); i++ {
		parent := views[i-1]
		s.refreshAgentNavigationRowsLocked(&parent)
		found := false
		for _, row := range parent.Rows {
			if row.CanOpen() && row.Address == views[i].Address {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		path = append(path, views[i].Address)
	}
	if n.view.Address != target {
		found := false
		for _, node := range s.agentCatalogTreeLocked() {
			if node.Row.Address == target && node.Row.CanOpen() {
				if _, ok := s.agentCatalogPathLocked(node.Path); ok {
					path = append(path, node.Path...)
					found = true
					break
				}
			}
		}
		if !found {
			return nil
		}
	}
	if !validAgentRestoreLineage(s.selectedID, target, path) {
		return nil
	}
	return append([]AgentSessionAddress(nil), path...)
}
func validAgentRestoreLineage(root string, target AgentSessionAddress, path []AgentSessionAddress) bool {
	if root == "" || len(path) == 0 || path[len(path)-1] != target {
		return false
	}
	parent := root
	seen := map[string]bool{root: true}
	for _, edge := range path {
		if !edge.validChild() || edge.ParentSessionID != parent || seen[edge.ChildSessionID] {
			return false
		}
		seen[edge.ChildSessionID] = true
		parent = edge.ChildSessionID
	}
	return true
}
