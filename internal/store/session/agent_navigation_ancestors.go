package session

type AgentNavigationAncestor struct {
	Depth            int
	SessionID, Title string
}

func (s *Store) AgentNavigationAncestors(token uint64) []AgentNavigationAncestor {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token {
		return nil
	}
	out := make([]AgentNavigationAncestor, 0, len(n.parents))
	for i, v := range n.parents {
		out = append(out, AgentNavigationAncestor{Depth: i, SessionID: v.SessionID, Title: v.Title})
	}
	return out
}

// Jump restores an already-owned ancestor snapshot in one operation. No follow,
// ordinary session selection, or child action is implicitly started.
func (s *Store) JumpAgentNavigation(token uint64, depth int) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token || depth < 0 || depth >= len(n.parents) {
		s.mu.Unlock()
		return
	}
	// Root remains reachable even if descendants were removed. Other ancestors
	// must still belong to the currently known parent chain.
	for i := 1; i <= depth; i++ {
		parent := n.parents[i-1]
		s.refreshAgentNavigationRowsLocked(&parent)
		target := n.parents[i]
		found := false
		for _, row := range parent.Rows {
			if row.CanOpen() && row.Address == target.Address && row.Address.ParentSessionID == parent.SessionID {
				found = true
				break
			}
		}
		if !found {
			s.mu.Unlock()
			return
		}
	}
	if n.cancel != nil {
		n.cancel()
		n.cancel = nil
	}
	s.cancelAgentCatalogReadsLocked()
	if n.actionCancel != nil {
		n.actionCancel()
		n.actionCancel = nil
	}
	n.actionEpoch++
	v := freezeAgentNavigationView(n.parents[depth])
	n.parents = n.parents[:depth]
	n.notifyPending = false
	v.Token = n.view.Token + 1
	v.Busy, v.ActionBusy = false, false
	n.view = v
	s.mu.Unlock()
	s.fire()
}
