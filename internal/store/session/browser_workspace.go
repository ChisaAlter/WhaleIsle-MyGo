package session

// BrowserWorkspaceScope snapshots the selected session's registered workspace
// (including inherited subagent membership), otherwise its Host cwd. It never
// resolves or reads a Host path on the desktop filesystem. Missing directory
// and Host scratch sessions use the fixed shared Browser scope.
func (s *Store) BrowserWorkspaceScope() (cwd, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.workspaceRowsLocked()
	cwd = s.cwd
	byID := make(map[string]Item, len(s.items))
	for _, it := range s.items {
		byID[it.SessionID] = it
	}
	member := false
	if selected, ok := byID[s.selectedID]; ok {
		cwd = selected.Cwd
		seen := map[string]bool{}
		for id := selected.SessionID; id != "" && !seen[id]; id = byID[id].ParentID {
			seen[id] = true
			matches := map[string]WorkspaceRow{}
			for _, row := range rows {
				for _, sid := range row.SessionIDs {
					if sid == id && row.Path != "" {
						matches[row.Path] = row
						member = true
						break
					}
				}
			}
			if len(matches) == 1 {
				for _, row := range matches {
					return row.Path, row.Title
				}
			}
			if len(matches) > 1 {
				break
			} // Ambiguous registry membership: retain the exact session cwd.
		}
	}
	if cwd == "" || (!member && s.scratchCwd != "" && cwd == s.scratchCwd) {
		return "", ""
	}
	for _, row := range rows {
		if row.Path == cwd {
			return row.Path, row.Title
		}
	}
	return cwd, cwd
}
