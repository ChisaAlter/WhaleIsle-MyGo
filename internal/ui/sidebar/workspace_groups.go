package sidebar

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func workspaceText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}
func workspaceViewOptions(c *ui.Context, m Model) {
	group := m.Snapshot.WorkspaceLayout.GroupBy
	if group == "" {
		group = "workspace"
	}
	order := m.Snapshot.WorkspaceLayout.OrderBy
	if order == "" {
		order = "manual"
	}
	groupLabels := []string{workspaceText("按工作区", "By workspace"), workspaceText("工作区树", "Workspace tree"), workspaceText("平铺会话", "Flat sessions")}
	groupValues := []string{"workspace", "workspace-tree", "flat"}
	selected := groupLabels[0]
	for i, v := range groupValues {
		if group == v {
			selected = groupLabels[i]
		}
	}
	ui.Select(c.Key("workspace-group-mode"), &selected, groupLabels).FillWidth().OnChange(func() {
		if m.OnWorkspaceGrouping != nil {
			for i, label := range groupLabels {
				if selected == label {
					m.OnWorkspaceGrouping(groupValues[i])
				}
			}
		}
	})
	orderLabels := []string{workspaceText("手动排序", "Manual order"), workspaceText("最近更新", "Recently updated")}
	selectedOrder := orderLabels[0]
	if order == "updated" {
		selectedOrder = orderLabels[1]
	}
	ui.Select(c.Key("workspace-order-mode"), &selectedOrder, orderLabels).FillWidth().OnChange(func() {
		if m.OnWorkspaceOrdering != nil {
			if selectedOrder == orderLabels[1] {
				m.OnWorkspaceOrdering("updated")
			} else {
				m.OnWorkspaceOrdering("manual")
			}
		}
	})
}
func workspaceRowsView(c *ui.Context, m Model, items []session.Item, fg, muted, border ui.Color) {
	groups := session.BuildWorkspaceGroups(m.Snapshot, items)
	layout := m.Snapshot.WorkspaceLayout
	hiddenParents := map[string]bool{}
	owners := session.WorkspaceSessionOwners(m.Snapshot.Workspaces, items)
	if layout.PreferenceError != "" {
		ui.Text(c, layout.PreferenceError).FontSize(theme.InterfaceSize(11)).TextColor(fg)
		if ui.Button(c, workspaceText("重试视图偏好", "Retry view preferences")).FontSize(theme.InterfaceSize(11)).Clicked() && m.OnRetryWorkspacePreferences != nil {
			m.OnRetryWorkspacePreferences()
		}
	}
	if layout.Error != "" {
		ui.Text(c, layout.Error).FontSize(theme.InterfaceSize(11)).TextColor(fg)
	}
	for _, group := range groups {
		g := group
		if hiddenParents[g.ParentID] {
			hiddenParents[g.Key] = true
			continue
		}
		collapsed := layout.Collapsed[g.Key]
		if m.SearchQuery != nil && strings.TrimSpace(*m.SearchQuery) != "" {
			collapsed = false
		}
		if g.Kind != "flat" {
			title := g.Workspace.Title
			if title == "" {
				title = g.Workspace.Path
			}
			if g.Kind == "scratch" {
				title = workspaceText("无目录会话", "No workspace folder")
			}
			if g.Kind == "other" {
				title = workspaceText("其他会话", "Other sessions")
			}
			header := ui.Row(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: g.Key})).FillWidth().Height(34).Gap(0).AlignItems(ui.Center).Padding(0, 8, 0, float32(8+g.Depth*12)).Radius(8)
			if header.Hovered() || header.FocusWithin() {
				header.Background(selectedBackground(theme.Resolve(m.ThemeID).Active(m.SystemDark)))
			}
			header.Children(func() {
				fold := "▾"
				if collapsed {
					fold = "▸"
				}
				if ui.Button(c, fold).Label(workspaceText("折叠工作区 ", "Toggle workspace ")+g.Key).Width(16).Height(20).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() && m.OnToggleWorkspaceGroup != nil {
					m.OnToggleWorkspaceGroup(g.Key)
				}
				titleButton := ui.Button(c, "").Label(title).Grow(1).MinWidth(0).Height(32).Padding(0, 6, 0, 6).BorderWidth(0).Background(ui.Transparent).TextColor(fg).Tooltip(g.Workspace.Path)
				titleButton.Children(func() {
					ui.Text(c, title).FontSize(theme.InterfaceSize(14)).TextColor(fg).MinWidth(0).Grow(1).SingleLine()
				})
				if g.Kind == "workspace" && !layout.Reordering {
					titleButton.Drag(workspaceDrag{HostGeneration: layout.HostGeneration, ID: g.Key})
				}
				if titleButton.Clicked() && g.Kind == "workspace" && m.OnSelectWorkspace != nil {
					m.OnSelectWorkspace(g.Key)
				}
				if g.Kind == "workspace" {
					beforeUp, up := session.WorkspaceMoveAnchor(groups, g.Key, -1)
					beforeDown, down := session.WorkspaceMoveAnchor(groups, g.Key, 1)
					more := ui.Button(c, "…").Label(workspaceText("工作区操作 ", "Workspace actions ") + g.Key).Width(16).Height(20).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
					more.Menu(func(menu *ui.Menu) {
						if m.OnManageWorkspace != nil && menu.Item(workspaceText("管理工作区", "Manage workspace")).Chosen() {
							m.OnManageWorkspace(g.Key)
						}
						if m.OnMoveWorkspaceBefore != nil {
							if menu.Item(workspaceText("上移工作区", "Move workspace up")).Disabled(!up || layout.Reordering).Chosen() {
								m.OnMoveWorkspaceBefore(g.Key, beforeUp)
							}
							if menu.Item(workspaceText("下移工作区", "Move workspace down")).Disabled(!down || layout.Reordering).Chosen() {
								m.OnMoveWorkspaceBefore(g.Key, beforeDown)
							}
						}
					})
					if !header.Hovered() && !header.FocusWithin() && !more.Focused() {
						more.Opacity(0)
					}
				} else if g.Kind == "scratch" && m.OnNoDirectory != nil && ui.Button(c, "+").Label(workspaceText("无目录会话", "No directory session")).Width(16).Height(20).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
					m.OnNoDirectory()
				}
			})
			workspaceDropTarget(header, m, g, groups, false, fg)
			if g.Kind == "workspace" {
				after := ui.Box(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "after/" + g.Key})).FillWidth().Height(5)
				workspaceDropTarget(after, m, g, groups, true, fg)
			}
		}
		if collapsed {
			hiddenParents[g.Key] = true
			continue
		}
		pins := m.Snapshot.PinnedSessionIDs
		if layout.OrderBy != "updated" {
			pins = sessionManualPins(g.Workspace.SessionIDs, pins)
		}
		rows := session.OrderedWorkspaceSessionRows(g.Items, g.Workspace.SessionIDs, pins, layout.OrderBy, g.Kind == "flat")
		for _, row := range rows {
			r := row
			rowModel := m
			rowModel.OnMoveSessionBefore = nil
			rowModel.sessionDrag = nil
			if g.Kind == "workspace" && layout.OrderBy != "updated" && !m.Batch.Enabled && m.OnMoveWorkspaceSessionBefore != nil {
				rowModel.sessionDrag = &sessionDragContext{Workspace: g.Key, Order: g.Workspace.SessionIDs, Rows: rows}
			}
			local := g.Kind == "flat" || g.Kind == "scratch" || g.Kind == "other"
			if local && layout.OrderBy != "updated" && !m.Batch.Enabled && m.OnMoveLocalSessionBefore != nil {
				rowModel.sessionDrag = &sessionDragContext{Local: true, Workspace: g.Key, Order: g.Workspace.SessionIDs, Rows: rows}
			}
			// A selected session in another workspace must reorder that workspace,
			// never whichever cwd happens to be selected for the next new conversation.
			owner := owners[r.Item.SessionID]
			if !local && owner != "" && layout.OrderBy != "updated" && m.OnMoveWorkspaceSessionBefore != nil {
				for _, ws := range m.Snapshot.Workspaces {
					if ws.WorkspaceID != owner {
						continue
					}
					for i, id := range ws.SessionIDs {
						if id == r.Item.SessionID && i > 0 {
							previous := ws.SessionIDs[i-1]
							rowModel.OnMoveSessionBefore = func(id, _ string) { m.OnMoveWorkspaceSessionBefore(owner, id, previous) }
						}
					}
				}
			}
			depth := r.Depth
			if g.Kind != "flat" {
				depth += g.Depth + 1
			}
			rowSession(c.Key(sessionRowKey{layout.HostGeneration, g.Key, r.Item.SessionID}), r.Item, depth, rowModel, fg, muted, border)
		}
	}
}
