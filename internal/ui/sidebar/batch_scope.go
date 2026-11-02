package sidebar

import "whaleisle/internal/store/session"

// BatchSelectionScope shares the browser's live model, including the common
// archive choice. Opening the independent archive section does not change it.
func BatchSelectionScope(m Model) session.SessionBatchScope {
	query := ""
	if m.SearchQuery != nil {
		query = *m.SearchQuery
	}
	return session.NewSessionBatchScope(!m.Collapsed, m.Snapshot.WorkspaceLayout.GroupBy, sharedArchiveFilter(m), query)
}
