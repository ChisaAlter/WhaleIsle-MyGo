package session

// SetWorkspaceArchivedFilter changes the one Host-scoped archive visibility
// choice shared by sidebar browsing and search. Expansion of the independent
// archived management section remains local UI state.
func (s *Store) SetWorkspaceArchivedFilter(filter string) {
	if filter != "default" && filter != "show" && filter != "only" {
		return
	}
	s.mu.Lock()
	s.workspaceLayout.view.ArchivedFilter = filter
	s.workspacePreferencesChangedLocked("archive", "")
	s.mu.Unlock()
	s.fire()
}
