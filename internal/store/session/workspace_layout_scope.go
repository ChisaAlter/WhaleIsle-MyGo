package session

// WorkspaceLayoutScope reuses the durable Host namespace already established
// for workspace navigation preferences. It contains no cookie, process pointer,
// connection generation or other restart-unstable credential.
func (s *Store) WorkspaceLayoutScope() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.workspacePreferences.scope
}
