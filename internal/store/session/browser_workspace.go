package session

// BrowserWorkspaceScope uses the authoritative Workspace membership, just as
// the desktop Browser source does. A session's cwd or parent is not evidence
// of membership; an ungrouped session retains its own persistent partition.
func (s *Store) BrowserWorkspaceScope() (cwd, title, sessionID string, ready bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionID, err = s.selectedID, s.workspaceErr
	if s.client == nil || !s.workspaceReady || err != nil {
		return "", "", sessionID, false, err
	}
	for _, row := range s.workspaceRowsLocked() {
		for _, member := range row.SessionIDs {
			if member == sessionID && sessionID != "" {
				return row.Path, row.Title, sessionID, true, nil
			}
		}
	}
	return "", "", sessionID, true, nil
}
