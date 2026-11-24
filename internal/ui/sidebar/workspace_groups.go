package sidebar

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

const flatSessionPageBase = 50

func sidebarPaginationToggle(c *ui.Context, key, label, accessible string, muted ui.Color) bool {
	return coss.Button(c, label, coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonSM,
		Key:     key,
	}).Label(accessible).FillWidth().Height(28).FontSize(fs(12)).TextColor(muted).BorderWidth(0).Clicked()
}

func workspaceText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func workspaceViewOptions(c *ui.Context, m Model) {
	items, run := workspaceViewOptionsItems(m)
	if chosen := coss.MenuPanel(c, items, coss.MenuOptions{MinWidth: 200}); chosen != "" {
		run(chosen)
	}
}
func visibleWorkspaceGroups(m Model, items []session.Item) []session.WorkspaceGroup {
	groups := session.BuildSidebarWorkspaceGroups(m.Snapshot, items)
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
	workspaceTitles := make(map[string]string, len(m.Snapshot.Workspaces))
	for _, workspace := range m.Snapshot.Workspaces {
		workspaceTitles[workspace.WorkspaceID] = session.WorkspaceDisplayTitle(workspace.Title, workspace.Path)
	}
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
	flatFill := len(groups) == 1 && groups[0].Kind == "flat" && m.sidebarBodyFill
	list := ui.Column(c.Key("workspace-drop-list")).FillWidth().Gap(design.Sp(1))
	if flatFill {
		list.Grow(1).MinHeight(0)
	} else {
		list.Shrink(0)
	}
	rowTransition := workspaceGroupListMotion(c, list, layout.HostGeneration, groups)
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
			block := ui.Column(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "group/" + g.Key})).FillWidth().Gap(design.Sp(0.5)).
				Transition(rowTransition)
			if flatFill {
				block.Grow(1).MinHeight(0)
			} else {
				block.Shrink(0)
			}
			blocks = append(blocks, workspaceDropBlock{Group: g, Element: block})
			block.Children(func() {
				t := tok(c)
				if g.Kind != "flat" {
					title := session.WorkspaceDisplayTitle(g.Workspace.Title, i18n.T(i18n.KeyWSDefaultWorkspaceTitle))
					if title == "" {
						title = g.Workspace.Path
					}
					if g.Kind == "scratch" {
						title = workspaceText("无工作目录", "No working directory")
					}
					if g.Kind == "pinned" {
						title = workspaceText("已置顶", "Pinned")
					}
					if g.Kind == "other" {
						title = workspaceText("其他会话", "Other sessions")
					}
					// The 34-DIP header carries the group's accessible name and
					// treeitem role, as the source's [role=treeitem] row does;
					// its inner buttons keep their own names for the actions.
					header := ui.Row(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: g.Key})).Role(ui.RoleTreeItem).Label(title).FillWidth().Height(34).Gap(0).AlignItems(ui.Center).Padding(0, design.Sp(2), 0, design.Sp(2)+float32(g.Depth)*design.Sp(3)).Radius(design.RadiusMD)
					if header.Hovered() || header.FocusWithin() || header.MenuOpen() {
						header.Background(sbSelectedBg(t))
					}
					holdsSelection := highlights[g.Key]
					anchor := workspaceHoverAnchor{row: header, workspace: g.Workspace, label: title, pressed: header.Pressed()}

					header.Children(func() {
						// Rows.module.css: the folder rests, and the row swaps it for the
						// disclosure arrow on hover (the arrow rotates as the group opens).
						hovered := header.Hovered() || header.FocusWithin()
						lead := coss.Button(c, "", coss.ButtonOptions{
							Variant: coss.ButtonGhost,
							Size:    coss.ButtonIconXS,
							Label:   workspaceText("折叠工作区 ", "Toggle workspace ") + g.Key,
							Content: func() {},
						}).Size(16, 20).Shrink(0).Padding(0).BorderWidth(0).Background(ui.Transparent).Center()
						lead.Children(func() {
							if hovered || g.Kind == "scratch" || g.Kind == "pinned" {
								size, ink := float32(12), muted
								if g.Kind == "scratch" || g.Kind == "pinned" {
									size, ink = design.IconXS, t.MutedForeground
									if holdsSelection {
										ink = t.Primary
									}
								}
								ui.Icon(c, workspaceChevronArtwork).Size(size, size).TextColor(ink).
									Label(workspaceText("展开箭头 ", "Workspace chevron ") + g.Key).
									Rotate(groupChevronAngle(c, lead, collapsed))
								return
							}
							ink := muted
							if holdsSelection {
								ink = t.Primary
							}
							folder := workspaceFolderClosedArtwork
							if !collapsed {
								folder = workspaceFolderOpenArtwork
							}
							ui.Icon(c, folder).Size(design.IconSM, design.IconSM).TextColor(ink).
								Label(workspaceText("工作区文件夹 ", "Workspace folder ") + g.Key)
						})
						if lead.Clicked() && m.OnToggleWorkspaceGroup != nil {
							resetWorkspaceWatermark(m, g.Key, groups)
							m.OnToggleWorkspaceGroup(g.Key)
						}
						titleButton := coss.Button(c, "", coss.ButtonOptions{
							Variant: coss.ButtonGhost,
							Size:    coss.ButtonSM,
							Label:   title,
							Content: func() {},
						}).Grow(1).MinWidth(0).Height(32).Padding(0, design.Sp(1.5), 0, design.Sp(1.5)).BorderWidth(0).Background(ui.Transparent).TextColor(fg)
						titleButton.Children(func() {
							text := ui.Text(c, title).FontSize(fs(14)).TextColor(fg).MinWidth(0).Grow(1).SingleLine()
							if g.Kind == "scratch" || g.Kind == "pinned" {
								text.FontSize(fs(12)).FontWeight(design.WeightMedium).TextColor(t.MutedForeground).FixedLineHeight(fs(18))
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
						if g.Kind != "pinned" {
							workspaceRowActions(c, m, header, g, muted)
						}
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
					var flatLimit *int
					flatHidden := 0
					revealIdx := -1
					if g.Kind == "flat" {
						flatLimit = ui.Local(block, workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "flat-limit"}, func() int { return flatSessionPageBase })
						if *flatLimit < flatSessionPageBase {
							*flatLimit = flatSessionPageBase
						}
						if m.searchState != nil && m.searchState.Reveal != "" {
							for idx, candidate := range rows {
								if candidate.Item.SessionID == m.searchState.Reveal {
									revealIdx = idx
									if idx >= *flatLimit {
										*flatLimit = ((idx / flatSessionPageBase) + 1) * flatSessionPageBase
									}
									break
								}
							}
						}
						if len(rows) > *flatLimit {
							flatHidden = len(rows) - *flatLimit
							rows = rows[:*flatLimit]
						}
					}
					renderRow := func(rc *ui.Context, r session.WorkspaceSessionRow) {
						rowModel := m
						rowModel.OnMoveSessionBefore = nil
						rowModel.sessionDrag = nil
						rowModel.pinnedSection = g.Kind == "pinned"
						if rowModel.pinnedSection {
							rowModel.sessionContext = workspaceTitles[owners[r.Item.SessionID]]
							if rowModel.sessionContext == "" {
								rowModel.sessionContext = workspaceText("无工作目录", "No working directory")
							}
						}
						if g.Kind == "workspace" && !m.Batch.Enabled && m.OnMoveWorkspaceSessionBefore != nil {
							rowModel.sessionDrag = &sessionDragContext{Workspace: g.Key, Order: g.Workspace.SessionIDs, Rows: rows}
						}
						if local && !m.Batch.Enabled && m.OnMoveLocalSessionBefore != nil {
							rowModel.sessionDrag = &sessionDragContext{Local: true, Workspace: g.Key, Order: g.Workspace.SessionIDs, Rows: rows}
						}
						// A selected session in another workspace must reorder that workspace,
						// never whichever cwd happens to be selected for the next new conversation.
						owner := owners[r.Item.SessionID]
						if !local && g.Kind != "pinned" && owner != "" && prevByWorkspace != nil {
							if previous, ok := prevByWorkspace[owner][r.Item.SessionID]; ok && previous != "" {
								rowModel.OnMoveSessionBefore = func(id, _ string) { m.OnMoveWorkspaceSessionBefore(owner, id, previous) }
							}
						}
						depth := r.Depth
						if g.Kind != "flat" {
							depth += g.Depth
						}
						rowSlot := ui.Column(rc.Key(sessionRowKey{layout.HostGeneration, g.Key, "slot/" + r.Item.SessionID})).FillWidth().Shrink(0).
							Transition(rowTransition)
						rowSlot.Children(func() {
							if isArchived(m.Snapshot, r.Item.SessionID) {
								title := r.Item.Title
								if strings.TrimSpace(title) == "" {
									title = i18n.T(i18n.KeySidebarNewSessionFallback)
								}
								rowArchivedSession(rc.Key(sessionRowKey{layout.HostGeneration, g.Key, r.Item.SessionID}), r.Item.SessionID, title, depth, rowModel, muted)
							} else {
								rowSession(rc.Key(sessionRowKey{layout.HostGeneration, g.Key, r.Item.SessionID}), r.Item, depth, rowModel, fg, muted, border)
							}
						})
					}
					if g.Kind == "flat" {
						if len(rows) > 0 {
							flatState := ui.Local(block, workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "flat-list-state"}, func() struct {
								list         ui.ListState
								revealSerial uint64
							} {
								return struct {
									list         ui.ListState
									revealSerial uint64
								}{}
							})
							if revealIdx >= 0 && revealIdx < len(rows) && m.searchState != nil && flatState.revealSerial != m.searchState.RevealSerial {
								flatState.list.ScrollTo(revealIdx, ui.Center)
								flatState.revealSerial = m.searchState.RevealSerial
							}
							capRatio := float32(1.0)
							if m.archiveExpanded != nil && *m.archiveExpanded {
								capRatio = 0.52
							}
							capH := sidebarListMaxHeight(m, c, capRatio)
							flatList := ui.List(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "flat-list"}), &flatState.list, len(rows)).
								ItemKey(func(i int) any { return sessionRowKey{layout.HostGeneration, g.Key, rows[i].Item.SessionID} }).
								FillWidth().MinWidth(0).Gap(design.Sp(0.5)).
								MaxHeight(capH).
								ScrollbarsVisible(m.scrollbarsVisible)
							if flatFill {
								flatList.Grow(1).MinHeight(0)
							} else {
								intrinsicH := float32(len(rows)*32 + max(0, len(rows)-1)*2)
								flatList.MinHeight(min(capH, intrinsicH))
							}
							flatList.Rows(func(lr ui.ListRow) {
								r := rows[lr.Index]
								if sidebarVirtualRowHook != nil {
									sidebarVirtualRowHook("flat", lr.Index, r.Item.SessionID)
								}
								renderRow(lr.Context, r)
							})
						}
					} else {
						for _, row := range rows {
							renderRow(c, row)
						}
					}
					if g.Kind == "flat" && flatLimit != nil && (flatHidden > 0 || (len(visible.Rows) > flatSessionPageBase && *flatLimit > flatSessionPageBase)) {
						label := i18n.WSf(i18n.KeyWSSessionsExpand, "n", strconv.Itoa(flatHidden))
						accessible := workspaceText("显示更多", "Show more") + " " + g.Key
						if flatHidden == 0 {
							label, accessible = i18n.T(i18n.KeyWSSessionsCollapse), workspaceText("收起会话", "Show less")+" "+g.Key
						}
						ui.Box(c).FillWidth().Shrink(0).Children(func() {
							if sidebarPaginationToggle(c, "flat-overflow/"+g.Key, label, accessible, muted) {
								if flatHidden == 0 {
									*flatLimit = flatSessionPageBase
								} else {
									*flatLimit += flatSessionPageBase
								}
							}
						})
					}
					if g.Kind != "flat" && (visible.Hidden > 0 || visible.Watermark > visible.Baseline) {
						label := i18n.WSf(i18n.KeyWSSessionsExpand, "n", strconv.Itoa(visible.Hidden))
						accessible := workspaceText("显示更多", "Show more") + " " + g.Key
						if visible.Hidden == 0 {
							label, accessible = i18n.T(i18n.KeyWSSessionsCollapse), workspaceText("收起会话", "Show less")+" "+g.Key
						}
						if sidebarPaginationToggle(c, "workspace-overflow/"+g.Key, label, accessible, muted) {
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
