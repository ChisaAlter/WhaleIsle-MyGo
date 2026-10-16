package sidebar

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

type sessionSearchRow struct {
	Item               session.Item
	Workspace, Snippet string
}

// deriveSessionSearchResults is a flat, bounded projection. Ordinary browsing
// order, pin order and parent/child expansion must not reorder content matches.
func deriveSessionSearchResults(snap session.ViewState, query string) ([]sessionSearchRow, bool) {
	query = strings.TrimSpace(query)
	if query == "" || !snap.WorkspaceReady {
		return nil, false
	}
	q := strings.ToLower(query)
	workspaces := map[string]string{}
	for _, ws := range snap.Workspaces {
		for _, id := range ws.SessionIDs {
			if _, seen := workspaces[id]; !seen {
				workspaces[id] = ws.Title
			}
		}
	}
	archived := map[string]bool{}
	for _, id := range snap.ArchivedSessionIDs {
		archived[id] = true
	}
	byID := map[string]session.Item{}
	var local []session.Item
	for _, it := range snap.Items {
		_, member := workspaces[it.SessionID]
		if it.SessionID == "" || it.Blank || it.Origin == "subagent" || archived[it.SessionID] || (!member && !snap.IsNoDirectory(it)) {
			continue
		}
		if _, seen := byID[it.SessionID]; seen {
			continue
		}
		byID[it.SessionID] = it
		if strings.Contains(strings.ToLower(strings.TrimSpace(it.Title)), q) || strings.Contains(strings.ToLower(workspaces[it.SessionID]), q) {
			local = append(local, it)
		}
	}
	sort.SliceStable(local, func(i, j int) bool {
		if local[i].UpdatedAt != local[j].UpdatedAt {
			return local[i].UpdatedAt > local[j].UpdatedAt
		}
		return local[i].SessionID < local[j].SessionID
	})
	ready := !snap.SearchLoading && !snap.SearchUseLocal && snap.SearchErr == "" && snap.SearchRPCQuery == query
	snippets := map[string]string{}
	if ready {
		for _, hit := range snap.SearchResults {
			if _, seen := snippets[hit.SessionId]; !seen {
				snippets[hit.SessionId] = hit.Snippet
			}
		}
	}
	seen := map[string]bool{}
	var rows []sessionSearchRow
	include := func(it session.Item) {
		if seen[it.SessionID] {
			return
		}
		seen[it.SessionID] = true
		rows = append(rows, sessionSearchRow{Item: it, Workspace: workspaces[it.SessionID], Snippet: snippets[it.SessionID]})
	}
	for _, it := range local {
		include(it)
	}
	if ready {
		for _, hit := range snap.SearchResults {
			if it, ok := byID[hit.SessionId]; ok {
				include(it)
			}
		}
	}
	more := ready && snap.SearchHasMore || len(rows) > session.SessionSearchResultLimit
	if len(rows) > session.SessionSearchResultLimit {
		rows = rows[:session.SessionSearchResultLimit]
	}
	return rows, more
}

// A nonempty query owns a flat results surface; ordinary tree/archive controls
// resume after clearing or opening a result. Existing selection handles drafts.
func sessionSearchResultsView(c *ui.Context, m Model, fg, muted ui.Color) bool {
	if m.SearchQuery == nil || strings.TrimSpace(*m.SearchQuery) == "" {
		return false
	}
	rows, more := deriveSessionSearchResults(m.Snapshot, *m.SearchQuery)
	if m.Snapshot.SearchLoading {
		ui.Text(c, workspaceText("正在搜索正文…", "Searching conversation content…")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if m.Snapshot.SearchErr != "" {
		ui.Text(c, workspaceText("正文搜索失败，仍显示标题和工作区匹配：", "Content search failed; title and workspace matches remain: ")+m.Snapshot.SearchErr).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if len(rows) == 0 && !m.Snapshot.SearchLoading {
		ui.Text(c, workspaceText("没有匹配的会话", "No matching sessions")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	for _, result := range rows {
		row := result
		if batchRow(c, m, row.Item.SessionID, row.Item.Title, fg) {
			continue
		}
		background := ui.Transparent
		if row.Item.SessionID == m.Snapshot.SelectedID {
			background = selectedBackground(theme.Resolve(m.ThemeID).Active(m.SystemDark))
		}
		button := ui.Button(c.Key(struct {
			Host uint64
			ID   string
		}{m.Snapshot.WorkspaceLayout.HostGeneration, row.Item.SessionID}), "").Label("Open search result "+row.Item.SessionID).FillWidth().BorderWidth(0).Background(background).Padding(6, 8, 6, 8)
		button.Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				title := strings.TrimSpace(row.Item.Title)
				if title == "" {
					title = workspaceText("未命名会话", "Untitled session")
				}
				if row.Item.Running {
					title = "● " + title
				}
				ui.Text(c, title).FontSize(theme.InterfaceSize(13)).TextColor(fg)
				label := row.Workspace
				if label == "" {
					label = workspaceText("无目录会话", "No workspace folder")
				}
				ui.Text(c, label).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if row.Snippet != "" {
					ui.Text(c, row.Snippet).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
			})
		})
		if button.Clicked() && m.OnSelect != nil {
			revealSessionSearchResult(m, row.Item.SessionID)
			*m.SearchQuery = ""
			m.OnSelect(row.Item.SessionID)
		}
	}
	if more {
		ui.Text(c, workspaceText("还有更多匹配，请缩小搜索范围。", "More matches exist. Refine your search.")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	return true
}

// Opening a hit returns to the normal tree. Reveal only its existing group
// chain, otherwise a collapsed workspace would immediately hide the result.
func revealSessionSearchResult(m Model, id string) {
	if m.OnToggleWorkspaceGroup == nil || m.Snapshot.WorkspaceLayout.GroupBy == "flat" {
		return
	}
	key := "::scratch"
	for _, ws := range m.Snapshot.Workspaces {
		for _, member := range ws.SessionIDs {
			if member == id {
				key = ws.WorkspaceID
				break
			}
		}
		if key != "::scratch" {
			break
		}
	}
	parents := map[string]string{}
	for _, group := range session.BuildWorkspaceGroups(m.Snapshot, m.Snapshot.Items) {
		parents[group.Key] = group.ParentID
	}
	var chain []string
	seen := map[string]bool{}
	for key != "" && !seen[key] {
		seen[key] = true
		chain = append(chain, key)
		key = parents[key]
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if m.Snapshot.WorkspaceLayout.Collapsed[chain[i]] {
			m.OnToggleWorkspaceGroup(chain[i])
		}
	}
}
