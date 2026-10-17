package session

import "fmt"

// AgentChatTarget is a durable routing value, never a session capability or a
// persisted reader. Capture it only from the current catalog or addressed child.
func (s *Store) AgentChatTarget(token uint64, address AgentSessionAddress) (string, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.agentNavigationOwnedLocked() || s.agentNavigation.view.Token != token || !address.validChild() {
		return "", "", false
	}
	v := s.agentNavigation.view
	if v.Address == address {
		return s.selectedID, v.Title, true
	}
	for _, row := range v.Rows {
		if row.Address == address {
			return s.selectedID, row.Label, true
		}
	}
	// Only an explicitly expanded, currently valid chain may expose a nested
	// routing address. This does not grant prompt/interrupt capability.
	for _, node := range s.agentCatalogTreeLocked() {
		if node.Row.Address == address && node.Row.CanOpen() {
			if _, valid := s.agentCatalogPathLocked(node.Path); valid {
				return s.selectedID, node.Row.Label, true
			}
		}
	}
	return "", "", false
}

// OpenAgentChat is an explicit user open/reconnect action. The caller owns an
// independent navigation pane. Host validates the direct-parent address; this
// never selects/promotes a child as an ordinary session or grants tool replies.
func (s *Store) OpenAgentChat(root string, address AgentSessionAddress, title string) bool {
	s.SyncAgentNavigationPane()
	s.mu.Lock()
	if s.agentNavigationOwner == nil || !s.agentPaneCurrentLocked() || !address.validChild() || root != s.selectedID || s.client == nil || s.mux == nil || s.planHistoryBinding != fmt.Sprintf("%p\x00%s", s.client, root) {
		s.mu.Unlock()
		return false
	}
	s.invalidateAgentNavigationLocked()
	token := s.agentNavigation.view.Token
	s.agentNavigation = agentNavigationState{
		view:    AgentNavigationView{Token: token, SessionID: address.ChildSessionID, Title: title, Address: address, Depth: 1},
		binding: s.planHistoryBinding, root: root, client: s.client, mux: s.mux, generation: s.followGeneration,
	}
	s.startAgentNavigationReadLocked(false)
	s.mu.Unlock()
	s.fire()
	return true
}
