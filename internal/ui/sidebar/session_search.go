package sidebar

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// sessionSearchState belongs to the retained sidebar root, so opening a result
// can collapse the input and reveal the same ordinary row after search unmounts.
// Backspacing the query to empty does not change Expanded.
type sessionSearchState struct {
	Expanded     bool
	Focus        bool
	Reveal       string
	RevealSerial uint64
}

// Workspace rendering retains the raised watermark before building this row.
// Consume the one-shot reveal after scrolling, so a later fold can reset it.
func revealSessionSearchRow(row ui.Element, m Model, id string) {
	if m.searchState == nil || m.searchState.Reveal != id {
		return
	}
	shown := ui.Local(row, "search-reveal", func() uint64 { return 0 })
	if *shown != m.searchState.RevealSerial {
		row.ScrollIntoView()
		*shown = m.searchState.RevealSerial
		m.searchState.Reveal = ""
	}
}

type sessionSearchRow struct {
	Archived           bool
	Item               session.Item
	Workspace, Snippet string
}

// deriveSessionSearchResults is a flat, bounded projection. Ordinary browsing
// order, pin order and parent/child expansion must not reorder content matches.
func deriveSessionSearchResults(snap session.ViewState, query string, filters ...string) ([]sessionSearchRow, bool) {
	query = session.NormalizeSessionSearchQuery(query)
	if query == "" || !snap.WorkspaceReady {
		return nil, false
	}
	filter := "default"
	if len(filters) > 0 {
		filter = filters[0]
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
		if it.Presentation != nil && it.Presentation.Owner != "" && it.Presentation.Owner != "whaleisle" {
			continue
		}
		_, member := workspaces[it.SessionID]
		if it.SessionID == "" || it.Blank || it.Origin == "subagent" || (!member && !snap.IsNoDirectory(it)) {
			continue
		}
		if !searchArchiveVisible(archived[it.SessionID], filter) {
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
		rows = append(rows, sessionSearchRow{Archived: archived[it.SessionID], Item: it, Workspace: workspaces[it.SessionID], Snippet: snippets[it.SessionID]})
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
	if m.SearchQuery == nil || session.NormalizeSessionSearchQuery(*m.SearchQuery) == "" {
		return false
	}
	if m.searchState != nil {
		m.searchState.Reveal = ""
	}
	workspaceRowsStatus(c, m, fg)
	rows, more := deriveSessionSearchResults(m.Snapshot, *m.SearchQuery, searchArchiveFilter(m))
	if m.Snapshot.SearchErr != "" {
		ui.Text(c, workspaceText("正文搜索失败，仍显示标题和工作区匹配：", "Content search failed; title and workspace matches remain: ")+m.Snapshot.SearchErr).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if len(rows) == 0 && !m.Snapshot.SearchLoading {
		ui.Text(c, workspaceText("无匹配会话", "No matching sessions")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	for _, result := range rows {
		row := result
		rowModel := workspaceOwnerRowModel(m, row.Item.SessionID)
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		batch := m.Batch.Enabled && !rowModel.ownerRow
		locked := batch && (m.Batch.Busy || m.Batch.Confirming || m.OnBatchEdit == nil)
		checkboxChanged := false
		button := ui.Button(c.Key(struct {
			Host uint64
			ID   string
		}{m.Snapshot.WorkspaceLayout.HostGeneration, row.Item.SessionID}), "").Label("Open search result "+row.Item.SessionID).FillWidth().MinHeight(48).BorderWidth(0).Radius(12).Background(ui.Transparent).Padding(4, 8).Disabled(rowModel.ownerRow && m.Batch.Enabled)
		selected := row.Item.SessionID == displayedSessionID(m)
		if batch {
			button.Label("Select session " + row.Item.SessionID).Disabled(locked || row.Item.Blank)
			_, selected = m.Batch.Selected[row.Item.SessionID]
		}
		menuOpen := button.MenuOpen()
		if selected || button.Hovered() || button.FocusWithin() || menuOpen {
			button.Background(selectedBackground(colors))
		}
		if !batch && !menuOpen {
			button.Tooltip(statusLabel(sessionStatus(m, row.Item)))
		}
		menu := func(menu *ui.Menu) {
			menu.CloseOnPointerLeave(true)
			if rowModel.OnUnarchive != nil && menu.Item(i18n.T(i18n.KeySidebarUnarchive)).Icon(menuUnarchiveArtwork).Chosen() {
				rowModel.OnUnarchive(row.Item.SessionID)
			}
			// The existing callback opens the confirmation, never the delete RPC.
			if rowModel.OnDelete != nil && menu.Item(i18n.T(i18n.KeySidebarDelete)).Icon(menuTrashArtwork).Danger(true).Chosen() {
				rowModel.OnDelete(row.Item.SessionID)
			}
			if rowModel.ownerRow {
				rowModel.PublicContributions.RenderMenu(menu, "sidebar.workspaces.session.menu.item", row.Item.SessionID)
			}
		}
		if row.Archived && !batch {
			button.ContextMenu(menu)
		}
		button.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(0).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Height(20).AlignItems(ui.Center).Children(func() {
					status := sessionStatus(m, row.Item)
					if row.Archived {
						status = ""
					}
					if batch {
						ui.Box(c).Size(16, 20).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
							check := ui.Checkbox(c.Key("batch-search-checkbox-"+row.Item.SessionID), &selected, "").Label("Select search checkbox "+row.Item.SessionID).Size(16, 16).Gap(0).Disabled(locked || row.Item.Blank)
							checkboxChanged = check.Changed()
						})
					} else {
						sessionStatusSlot(c, status, muted)
					}
					title := strings.TrimSpace(row.Item.Title)
					if title == "" {
						title = workspaceText("未命名会话", "Untitled session")
					}
					ink := fg
					if row.Archived {
						ink = muted
					}
					ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(ink).MinWidth(0).Grow(1).Margin(0, 0, 0, 4).SingleLine()
					publicSidebarSlot(c, rowModel, "sidebar.session.row.badge", row.Item.SessionID, nil)
					if row.Archived && !batch {
						more := ui.Button(c.Key("search-result-menu-"+row.Item.SessionID), "").Label(sessionActionsLabel(row.Item.SessionID, row.Item.Title)).Size(16, 16).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Center().Menu(menu)
						if !menuOpen {
							more.Tooltip(workspaceText("会话操作", "Session actions"))
						}
						more.Children(func() { ui.Icon(c, menuEllipsisArtwork).Size(16, 16).TextColor(muted) })
						showActions := button.Hovered() || button.FocusWithin() || button.MenuOpen()
						if !showActions && !more.Focused() {
							more.Absolute().Right(8).Opacity(0)
						}
					}
				})
				// The 20px status inset belongs outside the flex context: percentage
				// limits must resolve against the remaining metadata width.
				ui.Box(c).FillWidth().MinWidth(0).Height(17).Padding(0, 0, 0, 20).Children(func() {
					ui.Row(c).Label("Search result context " + row.Item.SessionID).FillWidth().MinWidth(0).Height(17).Gap(6).AlignItems(ui.Center).Children(func() {
						label := row.Workspace
						if label == "" {
							label = workspaceText("无目录会话", "No workspace folder")
						}
						workspace := ui.Text(c, label).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(17)).TextColor(muted).SingleLine().MinWidth(0).MaxWidthPercent(100)
						if row.Snippet != "" {
							workspace.MaxWidthPercent(40)
							ui.Text(c, row.Snippet).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(17)).TextColor(muted).SingleLine().Grow(1).MinWidth(0)
						}
					})
				})
			})
		})
		publicSidebarRowContext(rowModel, row.Item.SessionID, button.MenuOpen(), false)
		clicked := button.Clicked()
		if batch {
			if !locked && !row.Item.Blank && (checkboxChanged || clicked) {
				m.OnBatchEdit("select", []session.SessionBatchTarget{{ID: row.Item.SessionID, Title: row.Item.Title}})
			}
		} else if clicked && !row.Archived && rowModel.OnSelect != nil {
			revealSessionSearchResult(m, row.Item.SessionID)
			*m.SearchQuery = ""
			if m.searchState != nil {
				m.searchState.Expanded, m.searchState.Focus = false, false
				m.searchState.Reveal = row.Item.SessionID
				m.searchState.RevealSerial++
			}
			if m.SearchFocus != nil {
				*m.SearchFocus = false
			}
			rowModel.OnSelect(row.Item.SessionID)
		}
	}
	if more {
		ui.Text(c, fmt.Sprintf(workspaceText("仅显示前 %d 条结果，请缩小搜索范围。", "Only the first %d results are shown. Narrow your search."), session.SessionSearchResultLimit)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
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
		if session.WorkspaceGroupCollapsed(m.Snapshot, chain[i]) {
			m.OnToggleWorkspaceGroup(chain[i])
		}
	}
}

func searchArchiveFilter(m Model) string {
	return sharedArchiveFilter(m)
}
func searchArchiveVisible(archived bool, filter string) bool {
	switch filter {
	case "show":
		return true
	case "only":
		return archived
	default:
		return !archived
	}
}
