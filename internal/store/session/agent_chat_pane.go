package session

import "fmt"

// AgentChatTarget is a durable routing value, never a session capability or a
// persisted reader. Capture it only from the current catalog or addressed child.
func (s *Store) AgentChatTarget(token uint64, address AgentSessionAddress) (string, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.agentPaneCurrentLocked() || !s.agentNavigationOwnedLocked() || s.agentNavigation.view.Token != token || !address.validChild() {
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
	for _, parent := range s.agentNavigation.parents {
		s.refreshAgentNavigationRowsLocked(&parent)
		for _, row := range parent.Rows {
			if row.CanOpen() && row.Address == address {
				return s.selectedID, row.Label, true
			}
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

// OpenAgentChat is an explicit user open/reconnect action for a child directly
// owned by root. A persisted nested address has no authority to infer its
// ancestry, so nested addresses must use OpenAgentChatFromCatalog below.
// The caller owns an independent navigation pane. Host validates the
// direct-parent address; this never selects/promotes a child as an ordinary
// session or grants tool replies.
func (s *Store) OpenAgentChat(root string, address AgentSessionAddress, title string) bool {
	return s.openAgentChat(root, address, title, false)
}

// OpenAgentChatFromCatalog opens an explicitly selected nested child. The
// source occurrence must still expose the exact address under its current
// token, which proves every expanded parent edge before the destination pane
// acquires a read lease. This is the only path that may open a nested address;
// saved/legacy routing values alone are never treated as ancestry proof.
func (s *Store) OpenAgentChatFromCatalog(source *Store, token uint64, root string, address AgentSessionAddress, title string) bool {
	if source == nil || source == s || s.AgentNavigationRootStore() != source.AgentNavigationRootStore() {
		return false
	}
	owner, _, ok := source.AgentChatTarget(token, address)
	if !ok || owner != root {
		return false
	}
	return s.openAgentChat(root, address, title, true)
}

func (s *Store) openAgentChat(root string, address AgentSessionAddress, title string, allowNested bool) bool {
	if !allowNested && address.ParentSessionID != root {
		return false
	}
	if !s.prepareAgentReadReopen(root) {
		return false
	}
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
	s.retainAgentReadLocked(address)
	s.startAgentNavigationReadLocked(false)
	s.mu.Unlock()
	s.fire()
	return true
}
