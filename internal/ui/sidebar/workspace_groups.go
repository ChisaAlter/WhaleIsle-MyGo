package sidebar

import (
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
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
	workspaceViewOptionsMenu(c, m, nil)
}
func visibleWorkspaceGroups(m Model, items []session.Item) []session.WorkspaceGroup {
	groups := session.BuildWorkspaceGroups(m.Snapshot, items)
	if browseArchiveFilter(m) != "only" {
		if len(items) == 0 && (len(m.Snapshot.Workspaces) == 0 || m.Snapshot.WorkspaceLayout.GroupBy == "flat") {
			return nil
		}
		return groups
	}
	visible := make(map[string]bool, len(groups))
	parents := make(map[string]string, len(groups))
	for _, group := range groups {
		visible[group.Key] = len(group.Items) > 0
		parents[group.Key] = group.ParentID
	}
	var out []session.WorkspaceGroup
	for _, group := range groups {
		if !visible[group.Key] {
			continue
		}
		group.ParentID = ""
		group.Depth = 0
		for parent := parents[group.Key]; parent != ""; parent = parents[parent] {
			if visible[parent] {
				if group.ParentID == "" {
					group.ParentID = parent
				}
				group.Depth++
			}
		}
		out = append(out, group)
	}
	return out
}

func workspaceRowsView(c *ui.Context, m Model, items []session.Item, fg, muted, border ui.Color) {
	groups := visibleWorkspaceGroups(m, items)
	highlights := workspaceGroupHighlights(m, groups)
	layout := m.Snapshot.WorkspaceLayout
	hiddenParents := map[string]bool{}
	owners := session.WorkspaceSessionOwnersForView(m.Snapshot, items)
	var prevByWorkspace map[string]map[string]string
	if m.OnMoveWorkspaceSessionBefore != nil {
		prevByWorkspace = make(map[string]map[string]string, len(m.Snapshot.Workspaces))
		for _, ws := range m.Snapshot.Workspaces {
			if len(ws.SessionIDs) <= 1 {
				continue
			}
			mPrev := make(map[string]string, len(ws.SessionIDs)-1)
			for i := 1; i < len(ws.SessionIDs); i++ {
				mPrev[ws.SessionIDs[i]] = ws.SessionIDs[i-1]
			}
			prevByWorkspace[ws.WorkspaceID] = mPrev
		}
	}
	workspaceRowsStatus(c, m, fg)
	// Sections own their 4-DIP separation; the scroll container adds none.
	// Rows inside a section keep the reference's 2-DIP spacing.
	list := ui.Column(c.Key("workspace-drop-list")).FillWidth().Gap(4).Shrink(0)
	m = workspaceDragModel(list, m)
	blocks := make([]workspaceDropBlock, 0, len(groups))
	hoverState := workspaceHoverGroup(c, list, m)
	hoverAnchors := make([]workspaceHoverAnchor, 0, len(groups))
	list.Children(func() {
		for _, group := range groups {
			g := group
			if hiddenParents[g.ParentID] {
				hiddenParents[g.Key] = true
				continue
			}
			collapsed := session.WorkspaceGroupCollapsed(m.Snapshot, g.Key)
			if m.SearchQuery != nil && strings.TrimSpace(*m.SearchQuery) != "" {
				collapsed = false
			}
			block := ui.Column(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "group/" + g.Key})).FillWidth().Gap(2).Shrink(0)
			blocks = append(blocks, workspaceDropBlock{Group: g, Element: block})
			block.Children(func() {
				colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
				if g.Kind != "flat" {
					title := g.Workspace.Title
					if title == "" {
						title = g.Workspace.Path
					}
					if g.Kind == "scratch" {
						title = workspaceText("无工作目录", "No working directory")
					}
					if g.Kind == "other" {
						title = workspaceText("其他会话", "Other sessions")
					}
					header := ui.Row(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: g.Key})).FillWidth().Height(34).Gap(0).AlignItems(ui.Center).Padding(0, 8, 0, float32(8+g.Depth*12)).Radius(8)
					if header.Hovered() || header.FocusWithin() || header.MenuOpen() {
						header.Background(selectedBackground(colors))
					}
					holdsSelection := highlights[g.Key]
					anchor := workspaceHoverAnchor{row: header, workspace: g.Workspace, label: title, pressed: header.Pressed()}

					header.Children(func() {
						// Rows.module.css: the folder rests, and the row swaps it for the
						// disclosure arrow on hover (the arrow rotates as the group opens).
						hovered := header.Hovered() || header.FocusWithin()
						lead := ui.ButtonBase(c).Width(16).Height(20).Shrink(0).Padding(0).BorderWidth(0).
							Background(ui.Transparent).Label(workspaceText("折叠工作区 ", "Toggle workspace ") + g.Key).Center()
						lead.Children(func() {
							if hovered || g.Kind == "scratch" {
								size, ink := float32(12), muted
								if g.Kind == "scratch" {
									size, ink = 14, chrome.Color(colors.Muted)
									if holdsSelection {
										ink = chrome.Color(colors.Accent)
									}
								}
								ui.Icon(c, workspaceChevronArtwork).Size(size, size).TextColor(ink).
									Label(workspaceText("展开箭头 ", "Workspace chevron ") + g.Key).
									Rotate(groupChevronAngle(c, lead, collapsed))
								return
							}
							ink := muted
							if holdsSelection {
								ink = chrome.Color(colors.Accent)
							}
							folder := workspaceFolderClosedArtwork
							if !collapsed {
								folder = workspaceFolderOpenArtwork
							}
							ui.Icon(c, folder).Size(16, 16).TextColor(ink).
								Label(workspaceText("工作区文件夹 ", "Workspace folder ") + g.Key)
						})
						if lead.Clicked() && m.OnToggleWorkspaceGroup != nil {
							resetWorkspaceWatermark(m, g.Key, groups)
							m.OnToggleWorkspaceGroup(g.Key)
						}
						titleButton := ui.Button(c, "").Label(title).Grow(1).MinWidth(0).Height(32).Padding(0, 6, 0, 6).BorderWidth(0).Background(ui.Transparent).TextColor(fg)
						titleButton.Children(func() {
							text := ui.Text(c, title).FontSize(theme.InterfaceSize(14)).TextColor(fg).MinWidth(0).Grow(1).SingleLine()
							if g.Kind == "scratch" {
								text.FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(chrome.Color(colors.Muted)).FixedLineHeight(theme.InterfaceSize(18))
							}
						})
						if g.Kind == "workspace" && !layout.Reordering {
							workspaceDragSource(titleButton, m, g.Key)
						}
						if titleButton.Clicked() && m.OnToggleWorkspaceGroup != nil {
							resetWorkspaceWatermark(m, g.Key, groups)
							m.OnToggleWorkspaceGroup(g.Key)
						}
						anchor.pressed = anchor.pressed || lead.Pressed() || titleButton.Pressed()
						workspaceRowActions(c, m, header, g, muted)
					})
					if g.Kind == "workspace" && g.Workspace.CreatedAt != "" {
						hoverAnchors = append(hoverAnchors, anchor)
					}
					if header.Clicked() && m.OnToggleWorkspaceGroup != nil {
						resetWorkspaceWatermark(m, g.Key, groups)
						m.OnToggleWorkspaceGroup(g.Key)
					}
				}
				if collapsed {
					resetWorkspaceWatermark(m, g.Key, groups)
					hiddenParents[g.Key] = true
				}
				workspaceGroupSessionRun(c, block, m, g, collapsed, func(visible workspaceWatermarkRows) {
					rows := visible.Rows
					local := g.Kind == "flat" || g.Kind == "scratch" || g.Kind == "other"
					for _, row := range rows {
						r := row
						rowModel := m
						rowModel.OnMoveSessionBefore = nil
						rowModel.sessionDrag = nil
						if g.Kind == "workspace" && !m.Batch.Enabled && m.OnMoveWorkspaceSessionBefore != nil {
							rowModel.sessionDrag = &sessionDragContext{Workspace: g.Key, Order: g.Workspace.SessionIDs, Rows: rows}
						}
						if local && !m.Batch.Enabled && m.OnMoveLocalSessionBefore != nil {
							rowModel.sessionDrag = &sessionDragContext{Local: true, Workspace: g.Key, Order: g.Workspace.SessionIDs, Rows: rows}
						}
						// A selected session in another workspace must reorder that workspace,
						// never whichever cwd happens to be selected for the next new conversation.
						owner := owners[r.Item.SessionID]
						if !local && owner != "" && prevByWorkspace != nil {
							if previous, ok := prevByWorkspace[owner][r.Item.SessionID]; ok && previous != "" {
								rowModel.OnMoveSessionBefore = func(id, _ string) { m.OnMoveWorkspaceSessionBefore(owner, id, previous) }
							}
						}
						depth := r.Depth
						if g.Kind != "flat" {
							depth += g.Depth
						}
						if isArchived(m.Snapshot, r.Item.SessionID) {
							title := r.Item.Title
							if strings.TrimSpace(title) == "" {
								title = i18n.T(i18n.KeySidebarNewSessionFallback)
							}
							rowArchivedSession(c.Key(sessionRowKey{layout.HostGeneration, g.Key, r.Item.SessionID}), r.Item.SessionID, title, depth, rowModel, muted)
						} else {
							rowSession(c.Key(sessionRowKey{layout.HostGeneration, g.Key, r.Item.SessionID}), r.Item, depth, rowModel, fg, muted, border)
						}
					}
					if g.Kind != "flat" && (visible.Hidden > 0 || visible.Watermark > visible.Baseline) {
						label := workspaceText("展开其余 "+strconv.Itoa(visible.Hidden)+" 个会话", "Show "+strconv.Itoa(visible.Hidden)+" more sessions")
						accessible := workspaceText("显示更多", "Show more")
						if visible.Hidden == 0 {
							label, accessible = workspaceText("收起", "Show less"), workspaceText("收起会话", "Show less")
						}
						if ui.Button(c.Key("workspace-overflow/"+g.Key), label).Label(accessible + " " + g.Key).FillWidth().Height(28).FontSize(theme.InterfaceSize(12)).TextColor(muted).Background(ui.Transparent).BorderWidth(0).Clicked() {
							if visible.Hidden == 0 {
								resetWorkspaceWatermark(m, g.Key, nil)
							} else {
								retainWorkspaceWatermark(m, g.Key, visible.Watermark+workspaceWatermarkBase)
							}
						}
					}
				})
			})
		}
		workspaceDropGroupsTarget(list, m, blocks, fg)
	})
	registerWorkspaceHover(c, m, list, hoverState, hoverAnchors)
}
