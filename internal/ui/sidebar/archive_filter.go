package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
)

func normalizeArchiveFilter(filter string) string {
	if filter == "show" || filter == "only" {
		return filter
	}
	return "default"
}

func sharedArchiveFilter(m Model) string {
	// A store-backed model never consults the compatibility pointers, even
	// when a stale standalone binding was also passed by its caller.
	if m.OnArchivedFilter != nil || m.Snapshot.WorkspaceLayout.ArchivedFilter != "" {
		return normalizeArchiveFilter(m.Snapshot.WorkspaceLayout.ArchivedFilter)
	}
	if m.BrowseArchivedFilter != nil {
		return normalizeArchiveFilter(*m.BrowseArchivedFilter)
	}
	if m.SearchArchivedFilter != nil {
		return normalizeArchiveFilter(*m.SearchArchivedFilter)
	}
	return "default"
}

func canSetArchiveFilter(m Model) bool {
	return m.OnArchivedFilter != nil || m.BrowseArchivedFilter != nil || m.SearchArchivedFilter != nil
}

func setArchiveFilter(m Model, filter string) {
	if filter != "default" && filter != "show" && filter != "only" {
		return
	}
	if m.OnArchivedFilter != nil {
		m.OnArchivedFilter(filter)
		return
	}
	if m.BrowseArchivedFilter != nil {
		*m.BrowseArchivedFilter = filter
	}
	if m.SearchArchivedFilter != nil {
		*m.SearchArchivedFilter = filter
	}
}

func archiveViewOptions(c *ui.Context, m Model) {
	if !canSetArchiveFilter(m) {
		return
	}
	selected := sharedArchiveFilter(m)
	options := []coss.SelectOption{
		{Value: "default", Label: workspaceText("隐藏归档会话", "Hide archived sessions")},
		{Value: "show", Label: workspaceText("显示归档会话", "Show archived sessions")},
		{Value: "only", Label: workspaceText("仅归档会话", "Only archived sessions")},
	}
	sel := coss.Select(c, &selected, options, coss.SelectOptions{
		Label: "Archived session visibility",
		Key:   "workspace-archive-filter",
	}).FillWidth()
	if sel.Changed() {
		setArchiveFilter(m, selected)
	}
}
