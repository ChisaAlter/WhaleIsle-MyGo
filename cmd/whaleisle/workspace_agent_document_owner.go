package main

import (
	"whaleisle/internal/store/session"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
)

// Lookup never creates a pane or enters an address. A closed source cannot
// borrow another pane's token, even if that pane views the same child.
func (s *mainState) workspaceDocumentStore(source browserpreviewui.DocumentSource) *session.Store {
	s.mu.Lock()
	root := s.sessions
	s.mu.Unlock()
	if root == nil {
		return nil
	}
	if source.AddressKind == "subagent" && source.NavigationPaneID != "" {
		return root.AgentNavigationPaneByID(source.NavigationPaneID)
	}
	return root
}
