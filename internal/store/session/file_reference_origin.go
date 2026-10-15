package session

import "fmt"

// WorkspaceFileReferenceOrigin takes one locked snapshot of the active Host,
// session and its actual cwd. Configured workspace fallback is not file authority.
// The binding format matches the Files owner in cmd/whaleisle.
func (s *Store) WorkspaceFileReferenceOrigin() (binding, sessionID, cwd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.selectedID == "" {
		return "", "", ""
	}
	for _, item := range s.items {
		if item.SessionID == s.selectedID && item.Cwd != "" {
			return fmt.Sprintf("%p\x00%s", s.client, s.selectedID), s.selectedID, item.Cwd
		}
	}
	return "", "", ""
}
