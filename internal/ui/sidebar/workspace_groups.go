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

type groupedVirtualKind uint8

const (
	vrowGroupHeader groupedVirtualKind = iota
	vrowSession
	vrowGroupOverflow
)

type groupedVirtualRow struct {
	kind      groupedVirtualKind
	spanIndex int
	group     session.WorkspaceGroup
	collapsed bool
	topGap    bool
	row       session.WorkspaceSessionRow
	groupRows []session.WorkspaceSessionRow
	hidden    int
	watermark int
	baseline  int
}

func workspaceRowsView(c *ui.Context, m Model, items []session.Item, fg, muted, border ui.Color) {
	if m.schedulesBySession == nil && len(m.Schedules.Rows) > 0 {
		m.schedulesBySession = buildActiveScheduleMap(m.Schedules)
	}
	groups := visibleWorkspaceGroups(m, items)
	highlights := workspaceGroupHighlights(m, groups)
	layout := m.Snapshot.WorkspaceLayout
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
	flatHost := len(groups) == 1 && groups[0].Kind == "flat"
	list := ui.Column(c.Key("workspace-drop-list")).FillWidth().Gap(design.Sp(0.5))
	if m.sidebarBodyFill {
		list.Grow(1).MinHeight(0)
	} else {
		list.Shrink(0)
	}
	m = workspaceDragModel(list, m)
	hoverState := workspaceHoverGroup(c, list, m)
	hoverAnchors := make([]workspaceHoverAnchor, 0, len(groups))

	renderSessionRow := func(rc *ui.Context, g session.WorkspaceGroup, rows []session.WorkspaceSessionRow, r session.WorkspaceSessionRow) ui.Element {
		local := g.Kind == "flat" || g.Kind == "scratch" || g.Kind == "other"
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
		rowSlot := ui.Column(rc.Key(sessionRowKey{layout.HostGeneration, g.Key, "slot/" + r.Item.SessionID})).FillWidth().Shrink(0)
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
		return rowSlot
	}

	if flatHost {
		g := groups[0]
		list.Children(func() {
			block := ui.Column(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "group/" + g.Key})).FillWidth().Gap(design.Sp(0.5))
			if m.sidebarBodyFill {
				block.Grow(1).MinHeight(0)
			} else {
				block.Shrink(0)
			}
			block.Children(func() {
				visible := visibleWorkspaceSessionRows(m, g)
				rows := visible.Rows
				flatLimit := ui.Local(block, workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "flat-limit"}, func() int { return flatSessionPageBase })
				if *flatLimit < flatSessionPageBase {
					*flatLimit = flatSessionPageBase
				}
				revealIdx := -1
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
				flatHidden := 0
				if len(rows) > *flatLimit {
					flatHidden = len(rows) - *flatLimit
					rows = rows[:*flatLimit]
				}
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
					if m.sidebarBodyFill {
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
						renderSessionRow(lr.Context, g, rows, r)
					})
				}
				if flatHidden > 0 || (len(visible.Rows) > flatSessionPageBase && *flatLimit > flatSessionPageBase) {
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
			})
		})
		registerWorkspaceHover(c, m, list, hoverState, hoverAnchors)
		return
	}

	// Flatten visible workspace groups, session rows, and per-group pagination
	// toggles into a single virtualized ui.List so large workspaces only build
	// the rows currently inside the viewport.
	hiddenParents := map[string]bool{}
	spans := make([]workspaceGroupSpan, 0, len(groups))
	vrows := make([]groupedVirtualRow, 0, len(groups)*4)
	revealIdx := -1
	revealID := ""
	if m.searchState != nil {
		revealID = m.searchState.Reveal
	}
	var intrinsicH float32
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
		spanIdx := len(spans)
		spans = append(spans, workspaceGroupSpan{group: g})
		topGap := spanIdx > 0
		vrows = append(vrows, groupedVirtualRow{
			kind:      vrowGroupHeader,
			spanIndex: spanIdx,
			group:     g,
			collapsed: collapsed,
			topGap:    topGap,
		})
		intrinsicH += 34
		if topGap {
			intrinsicH += 2
		}
		if collapsed {
			resetWorkspaceWatermark(m, g.Key, groups)
			hiddenParents[g.Key] = true
			continue
		}
		visible := visibleWorkspaceSessionRows(m, g)
		retainWorkspaceWatermark(m, g.Key, visible.Watermark)
		for _, r := range visible.Rows {
			if revealID != "" && revealIdx < 0 && r.Item.SessionID == revealID {
				revealIdx = len(vrows)
			}
			vrows = append(vrows, groupedVirtualRow{
				kind:      vrowSession,
				spanIndex: spanIdx,
				group:     g,
				row:       r,
				groupRows: visible.Rows,
			})
			intrinsicH += 32
		}
		if visible.Hidden > 0 || visible.Watermark > visible.Baseline {
			vrows = append(vrows, groupedVirtualRow{
				kind:      vrowGroupOverflow,
				spanIndex: spanIdx,
				group:     g,
				hidden:    visible.Hidden,
				watermark: visible.Watermark,
				baseline:  visible.Baseline,
			})
			intrinsicH += 28
		}
	}
	if len(vrows) > 1 {
		intrinsicH += float32(len(vrows)-1) * 2
	}

	list.Children(func() {
		if len(vrows) == 0 {
			return
		}
		var rowTransition ui.ElementTransition
		if !m.sidebarBodyFill {
			rowTransition = workspaceGroupListMotion(c, list, layout.HostGeneration, groups)
		}
		renderVirtualRow := func(rc *ui.Context, index int) {
			vr := vrows[index]
			g := vr.group
			switch vr.kind {
			case vrowGroupHeader:
				if sidebarVirtualRowHook != nil {
					sidebarVirtualRowHook("workspace", index, "group:"+g.Key)
				}
				t := tok(rc)
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
				// treeitem role; non-first headers add 2 DIP top margin so
				// collapsed group pitch matches the reference 38 DIP (34+2+2).
				header := ui.Row(rc.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: g.Key})).
					Role(ui.RoleTreeItem).Label(title).FillWidth().Height(34).Gap(0).AlignItems(ui.Center).
					Padding(0, design.Sp(2), 0, design.Sp(2)+float32(g.Depth)*design.Sp(3)).Radius(design.RadiusMD)
				if vr.topGap {
					header.Margin(design.Sp(0.5), 0, 0, 0)
				}
				if !m.sidebarBodyFill {
					header.Transition(rowTransition)
				}
				if header.Hovered() || header.FocusWithin() || header.MenuOpen() {
					header.Background(sbHoverBg(t))
				}
				holdsSelection := highlights[g.Key]
				anchor := workspaceHoverAnchor{row: header, workspace: g.Workspace, label: title, pressed: header.Pressed()}
				collapsed := vr.collapsed
				header.Children(func() {
					hovered := header.Hovered() || header.FocusWithin()
					lead := coss.Button(rc, "", coss.ButtonOptions{
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
							groupChevronIcon(rc, lead, collapsed, size, ink, workspaceText("展开箭头 ", "Workspace chevron ")+g.Key)
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
						ui.Icon(rc, folder).Size(design.IconSM, design.IconSM).TextColor(ink).
							Label(workspaceText("工作区文件夹 ", "Workspace folder ") + g.Key)
					})
					if lead.Clicked() && m.OnToggleWorkspaceGroup != nil {
						resetWorkspaceWatermark(m, g.Key, groups)
						m.OnToggleWorkspaceGroup(g.Key)
					}
					titleButton := coss.Button(rc, "", coss.ButtonOptions{
						Variant: coss.ButtonGhost,
						Size:    coss.ButtonSM,
						Label:   title,
						Content: func() {},
					}).Grow(1).MinWidth(0).Height(32).Padding(0, design.Sp(1.5), 0, design.Sp(1.5)).BorderWidth(0).Background(ui.Transparent).TextColor(fg)
					titleButton.Children(func() {
						text := ui.Text(rc, title).FontSize(fs(14)).TextColor(fg).MinWidth(0).Grow(1).SingleLine()
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
						workspaceRowActions(rc, m, header, g, muted)
					}
				})
				if g.Kind == "workspace" && g.Workspace.CreatedAt != "" {
					hoverAnchors = append(hoverAnchors, anchor)
				}
				if header.Clicked() && m.OnToggleWorkspaceGroup != nil {
					resetWorkspaceWatermark(m, g.Key, groups)
					m.OnToggleWorkspaceGroup(g.Key)
				}
				spans[vr.spanIndex].include(header)

			case vrowSession:
				if sidebarVirtualRowHook != nil {
					sidebarVirtualRowHook("workspace", index, vr.row.Item.SessionID)
				}
				el := renderSessionRow(rc, g, vr.groupRows, vr.row)
				spans[vr.spanIndex].include(el)

			case vrowGroupOverflow:
				if sidebarVirtualRowHook != nil {
					sidebarVirtualRowHook("workspace", index, "overflow:"+g.Key)
				}
				label := i18n.WSf(i18n.KeyWSSessionsExpand, "n", strconv.Itoa(vr.hidden))
				accessible := workspaceText("显示更多", "Show more") + " " + g.Key
				if vr.hidden == 0 {
					label, accessible = i18n.T(i18n.KeyWSSessionsCollapse), workspaceText("收起会话", "Show less")+" "+g.Key
				}
				slot := ui.Box(rc).FillWidth().Shrink(0)
				slot.Children(func() {
					if sidebarPaginationToggle(rc, "workspace-overflow/"+g.Key, label, accessible, muted) {
						if vr.hidden == 0 {
							resetWorkspaceWatermark(m, g.Key, nil)
						} else {
							retainWorkspaceWatermark(m, g.Key, vr.watermark+workspaceWatermarkBase)
						}
					}
				})
				spans[vr.spanIndex].include(slot)
			}
		}

		if m.sidebarBodyFill {
			vstate := ui.Local(list, workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "workspace-vlist-state"}, func() struct {
				list         ui.ListState
				revealSerial uint64
			} {
				return struct {
					list         ui.ListState
					revealSerial uint64
				}{}
			})
			if revealIdx >= 0 && revealIdx < len(vrows) && m.searchState != nil && vstate.revealSerial != m.searchState.RevealSerial {
				vstate.list.ScrollTo(revealIdx, ui.Center)
				vstate.revealSerial = m.searchState.RevealSerial
			}
			capRatio := float32(1.0)
			if m.archiveExpanded != nil && *m.archiveExpanded {
				capRatio = 0.52
			}
			capH := sidebarListMaxHeight(m, c, capRatio)
			vlist := ui.List(c.Key(workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "workspace-vlist"}), &vstate.list, len(vrows)).
				ItemKey(func(i int) any {
					vr := vrows[i]
					switch vr.kind {
					case vrowGroupHeader:
						return workspaceRowKey{HostGeneration: layout.HostGeneration, ID: vr.group.Key}
					case vrowGroupOverflow:
						return workspaceRowKey{HostGeneration: layout.HostGeneration, ID: "overflow/" + vr.group.Key}
					default:
						return sessionRowKey{layout.HostGeneration, vr.group.Key, vr.row.Item.SessionID}
					}
				}).
				FillWidth().MinWidth(0).Grow(1).MinHeight(0).Gap(design.Sp(0.5)).
				MaxHeight(capH).
				ScrollbarsVisible(m.scrollbarsVisible)
			vlist.Rows(func(lr ui.ListRow) {
				renderVirtualRow(lr.Context, lr.Index)
			})
		} else {
			for i := range vrows {
				renderVirtualRow(c, i)
			}
		}
		workspaceDropVirtualGroupsTarget(list, m, spans, fg)
	})
	registerWorkspaceHover(c, m, list, hoverState, hoverAnchors)
}
