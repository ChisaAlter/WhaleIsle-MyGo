package sidebar

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
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
	if layout.Error != "" {
		ui.Text(c, layout.Error).FontSize(11).TextColor(fg)
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
			ui.Row(c.Key("workspace-header-"+g.Key)).FillWidth().Gap(4).AlignItems(ui.Center).Padding(4, 0, 4, float32(g.Depth*12)).Children(func() {
				fold := "▾"
				if collapsed {
					fold = "▸"
				}
				if ui.Button(c, fold).Width(20).BorderWidth(0).Background(ui.Transparent).Clicked() && m.OnToggleWorkspaceGroup != nil {
					m.OnToggleWorkspaceGroup(g.Key)
				}
				if ui.Button(c, title).Grow(1).FontSize(12).BorderWidth(0).Background(ui.Transparent).TextColor(fg).Tooltip(g.Workspace.Path).Clicked() && g.Kind == "workspace" && m.OnSelectWorkspace != nil {
					m.OnSelectWorkspace(g.Key)
				}
				if g.Kind == "workspace" {
					beforeUp, up := session.WorkspaceMoveAnchor(groups, g.Key, -1)
					beforeDown, down := session.WorkspaceMoveAnchor(groups, g.Key, 1)
					if ui.Button(c, "↑").FontSize(10).Disabled(!up || layout.Reordering).Tooltip(workspaceText("上移工作区", "Move workspace up")).Clicked() && m.OnMoveWorkspaceBefore != nil {
						m.OnMoveWorkspaceBefore(g.Key, beforeUp)
					}
					if ui.Button(c, "↓").FontSize(10).Disabled(!down || layout.Reordering).Tooltip(workspaceText("下移工作区", "Move workspace down")).Clicked() && m.OnMoveWorkspaceBefore != nil {
						m.OnMoveWorkspaceBefore(g.Key, beforeDown)
					}
					if ui.Button(c, "⋯").FontSize(11).Tooltip(workspaceText("管理工作区", "Manage workspace")).Clicked() && m.OnManageWorkspace != nil {
						m.OnManageWorkspace(g.Key)
					}
				} else if g.Kind == "scratch" && m.OnNoDirectory != nil && ui.Button(c, "＋").FontSize(11).Clicked() {
					m.OnNoDirectory()
				}
			})
		}
		if collapsed {
			hiddenParents[g.Key] = true
			continue
		}
		rows := session.OrderedWorkspaceSessionRows(g.Items, g.Workspace.SessionIDs, m.Snapshot.PinnedSessionIDs, layout.OrderBy, g.Kind == "flat")
		for _, row := range rows {
			r := row
			rowModel := m
			rowModel.OnMoveSessionBefore = nil
			// A selected session in another workspace must reorder that workspace,
			// never whichever cwd happens to be selected for the next new conversation.
			owner := owners[r.Item.SessionID]
			if owner != "" && layout.OrderBy != "updated" && m.OnMoveWorkspaceSessionBefore != nil {
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
			rowSession(c.Key("workspace-session-"+r.Item.SessionID), r.Item, depth, rowModel, fg, muted, border)
		}
	}
}
