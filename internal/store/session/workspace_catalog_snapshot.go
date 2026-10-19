package session

import "whaleisle/internal/dsh"

// CachedWorkspaceChoices uses the already-followed registry for this exact Host
// client. It neither lists directories nor creates or activates a workspace.
func (s *Store) CachedWorkspaceChoices(client *dsh.Client) ([]WorkspaceRow, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client == nil || s.client != client || !s.workspaceReady {
		return nil, false
	}
	return s.workspaceRowsLocked(), s.workspaceReady
}
