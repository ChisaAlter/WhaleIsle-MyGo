// Package sidebar draws the workspace and session tree stub for M4.
package sidebar

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
)

// viewport is the window's client area in DIPs: the frame's measurement when the
// sidebar is painted inside it, Context.Size otherwise (tests, previews).
func (m Model) viewport(c *ui.Context) (width, height float32) {
	if m.ViewportW > 0 && m.ViewportH > 0 {
		return m.ViewportW, m.ViewportH
	}
	return c.Size()
}

// Model is sidebar state for one frame.
type Model struct {
	// ViewportW/H are the window's client area in DIPs. Context.Size reports
	// device pixels on a scaled display, so the hover cards take their bounds
	// from here and fall back to Context.Size only when the frame did not say.
	ViewportW                    float32
	ViewportH                    float32
	ConnectionReady              bool
	ConnectionError              string
	DesktopUpdateInstalling      bool
	DesktopUpdatePhase           string
	DesktopUpdateDetail          string
	OnOpenDesktopUpdate          func()
	ConnectionConnecting         bool
	ConnectionEpoch              uint64
	OnReconnect                  func()
	PendingStatuses              map[string]string
	sessionDrag                  *sessionDragContext
	Batch                        session.SessionBatch
	OnBatchEdit                  func(string, []session.SessionBatchTarget)
	OnBatchRequest               func(string)
	OnBatchConfirm               func(uint64)
	OnBatchCancel                func(uint64)
	Schedules                    schedstore.ViewState
	OnRetryWorkspacePreferences  func()
	OnDropWorkspaceBefore        func(uint64, string, string)
	OnSelectWorkspace            func(string)
	OnManageWorkspace            func(string)
	OnWorkspaceGrouping          func(string)
	OnWorkspaceOrdering          func(string)
	OnToggleWorkspaceGroup       func(string)
	OnMoveWorkspaceBefore        func(string, string)
	OnMoveWorkspaceSessionBefore func(string, string, string)
	OnMoveLocalSessionBefore     func(uint64, string, string, string)

	Projects         projects.Model
	HideArchivedList bool
	// Account is the bottom account seat (avatar, name, menu). The shell fills
	// it from the account details snapshot and the sign-out flow; the zero value
	// still renders the row and routes account settings to OnOpenSettings.
	Account Account
	// BrowseArchivedFilter is a Host-local list filter, independent of search.
	BrowseArchivedFilter *string
	Collapsed            bool
	OnToggle             func()
	Snapshot             session.ViewState
	ThemeID              string
	SystemDark           bool
	SearchQuery          *string
	// SearchArchivedFilter is a search-local visibility choice: default, show, only.
	SearchArchivedFilter *string
	SearchFocus          *bool
	WorkspaceLabel       string
	OnSelect             func(sessionID string)
	OnRefresh            func()
	OnCreate             func()
	OnNoDirectory        func()
	OnOpenMarket         func()
	OnOpenSchedule       func()
	OnOpenAssistant      func()
	OnOpenLauncher       func()
	OnContactAccount     func()
	OnOpenSettings       func()
	// OnBrowseWorkspace opens the folder picker (SidebarRoot ☰ / workspace row).
	OnBrowseWorkspace func()
	// OnRevealWorkspace opens the current workspace in the OS file manager.
	OnRevealWorkspace func()
	// OnRenameWorkspace calls workspace/rename when a workspace id is known.
	OnRenameWorkspace func(title string)
	// OnDeleteWorkspace calls workspace/delete when a workspace id is known.
	OnDeleteWorkspace    func()
	WorkspaceRenameDraft *string
	// WorkspaceMenuOpen toggles the ☰ popover; nil uses a package fallback.
	WorkspaceMenuOpen    *bool
	RenameSessionID      string
	RenameDraft          *string
	RenameBusy           bool
	RenameError          string
	OnRenameDraftChanged func(string)
	OnBeginRename        func(sessionID, currentTitle string)
	OnCommitRename       func(sessionID, title string)
	OnCancelRename       func()
	OnArchive            func(sessionID string)
	OnUnarchive          func(sessionID string)
	OnDelete             func(sessionID string)
	OnPin                func(sessionID string)
	OnUnpin              func(sessionID string)
	OnCopySessionID      func(sessionID string)
	OnFork               func(sessionID string)
	// OnMoveSessionBefore reorders via workspace/insertSessionBefore (beforeSessionID may be empty = top).
	OnMoveSessionBefore func(sessionID, beforeSessionID string)
}

var workspaceMenuFB bool

type treeNode struct {
	item     session.Item
	children []treeNode
}

// View paints the left session tree.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	// SidebarRoot.root: flex column height 100%, pad 6 / inline 12.
	// Children (flex:none): logoRow (60), newSession, panelList;
	// regionArea flex:1 min-height:0; footArea flex:none (pinned bottom).
	root := ui.Column(c.Key("sidebar-root")).Fill().MinWidth(0).Padding(6, 12, 6, 12)
	if m.Collapsed {
		root.Padding(18, 10, 6, 10)
	}
	if m.SearchFocus == nil {
		m.SearchFocus = ui.Local(root, "search-focus-request", func() bool { return false })
	}
	if m.BrowseArchivedFilter == nil {
		m.BrowseArchivedFilter = ui.Local(root, struct {
			Key  string
			Host uint64
		}{"browse-archive-filter", m.Snapshot.WorkspaceLayout.HostGeneration}, func() string { return "default" })
	}
	if m.SearchArchivedFilter == nil {
		m.SearchArchivedFilter = ui.Local(root, struct {
			Key  string
			Host uint64
		}{"search-archive-filter", m.Snapshot.WorkspaceLayout.HostGeneration}, func() string { return "default" })
	}
	root.Draw(chrome.Glass(chrome.Color(colors.Sidebar), chrome.MaterialSidebar)).
		Children(func() {
			if m.Collapsed {
				// SidebarRoot.railIn: the rail's controls enter from the former
				// expanded edge over 150 ms as the track finishes collapsing.
				rail := ui.Column(c.Key("sidebar-rail-content")).FillWidth().Grow(1).MinHeight(0)
				if !c.Preferences().ReduceMotion {
					rail.Transition(ui.ElementTransition{
						Duration: 150 * time.Millisecond,
						Ease:     ui.EaseOut,
						Position: true,
						Enter:    &ui.Motion{X: 49, Opacity: 0},
					})
				}
				rail.Children(func() { collapsedRail(c, m, fg, colors) })
			} else {
				// SidebarRoot.module.css .logoRow: 60px tall (40 under a native
				// caption), and the original brand lock: 38px mark, 鲸屿 18/700
				// with the 屿 glyph in the brand accent, MYGO鲸屿 10/600, and the
				// 8/500 Harness attribution (Brand.module.css).
				ui.Row(c).Key("sidebar-brand-row").FillWidth().Height(60).Gap(8).AlignItems(ui.Center).
					DragWindow().Children(func() {
					if bmp := chrome.BrandAvatar(); bmp != nil {
						ui.Image(c, bmp).Size(38, 38).Fit(ui.Contain)
					} else {
						ui.Box(c).Size(38, 38).Radius(19).Background(chrome.Color(colors.Accent).Alpha(0.25))
					}
					ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
						ui.Row(c).Height(theme.InterfaceSize(22)).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Row(c).Gap(0).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "鲸").FontSize(theme.InterfaceSize(18)).FixedLineHeight(theme.InterfaceSize(22)).
									FontWeight(700).LetterSpacing(-0.7).TextColor(fg)
								ui.Text(c, "屿").FontSize(theme.InterfaceSize(18)).FixedLineHeight(theme.InterfaceSize(22)).
									FontWeight(700).LetterSpacing(-0.7).TextColor(chrome.Color(colors.Accent))
							})
							// Baselines line up: the 10px wordmark sits on the 18px
							// line's baseline, as the original's titleRow does.
							ui.Text(c, "WHALE ISLE").FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(12)).
								FontWeight(600).LetterSpacing(1.2).TextColor(fg).Padding(7, 0, 0, 0)
						})
						ui.Text(c, "BASED ON DEEPSEEK HARNESS").FontSize(theme.InterfaceSize(8)).
							FixedLineHeight(theme.InterfaceSize(10)).FontWeight(500).LetterSpacing(0.32).TextColor(muted)
					})
					if m.OnToggle != nil {
						collapse := ui.Button(c, "").Label(i18n.T(i18n.KeySidebarCollapse)).Tooltip(i18n.T(i18n.KeySidebarCollapse) + " (Ctrl+B)").Width(24).Height(24).BorderWidth(0).Background(ui.Transparent)
						collapse.Draw(func(p *ui.Painter, r ui.Rect) {
							// Reference chrome: sidebar panel glyph, not a chevron.
							p.Stroke(ui.Rect{X: r.X + 5, Y: r.Y + 6, W: 14, H: 12}, fg, 1, 2)
							path := &ui.Path{}
							path.MoveTo(r.X+10.5, r.Y+6)
							path.LineTo(r.X+10.5, r.Y+18)
							p.StrokePath(path, 1, fg)
						})
						if collapse.Clicked() {
							m.OnToggle()
						}
					}
				})
				if m.OnCreate != nil {
					// SidebarRoot.module.css .newSession: h38, radius 12, margin 0 2 8, border-l2.
					btn := ui.Button(c, "").Label(i18n.T(i18n.KeySidebarNewSessionFallback)).FillWidth().Height(38).Radius(12).
						Background(chrome.Color(colors.Field)).Border(1, border).
						TextColor(fg).FontSize(theme.InterfaceSize(14)).FontWeight(500).Padding(8, 16, 8, 16)
					btn.Children(func() {
						ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Gap(8).Children(func() {
							plus := ui.Box(c).Size(18, 18).Radius(9).Border(1, fg.Alpha(.55)).Center()
							plus.Children(func() {
								ui.Icon(c, ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M8 3v10M3 8h10"/></svg>`))).Size(10, 10).TextColor(fg)
							})
							ui.Text(c, i18n.T(i18n.KeySidebarNewSessionFallback)).
								FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg).SingleLine()
						})
					})
					if btn.Clicked() {
						m.OnCreate()
					}
					ui.Box(c).Height(8)
				}
				// SidebarRoot.panelList: .panelRow is 36 high and the rows butt together, so
				// the pitch is 36 (the pass-2 measurement; a 4-DIP gap made it 40).
				ui.Column(c).FillWidth().Padding(0, 0, 8, 0).Children(func() {
					nav := func(drawIcon func(), label string, fn func()) {
						row := ui.Button(c, "").FillWidth().Height(36).Radius(12).
							Background(ui.Transparent).BorderWidth(0).
							Padding(7, 8, 7, 8).AlignItems(ui.Center)
						row = row.Transition(ui.ElementTransition{Duration: chrome.PopoverDuration, Colors: true})
						row.Children(func() {
							ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
								if drawIcon != nil {
									drawIcon()
								}
								ui.Text(c, label).FontSize(theme.InterfaceSize(14)).TextColor(fg).Grow(1)
							})
						})
						if fn != nil && row.Clicked() {
							fn()
						}
					}
					nav(func() {
						ink := muted
						ui.Box(c).Width(16).Height(16).Draw(func(p *ui.Painter, r ui.Rect) {
							path := &ui.Path{}
							path.MoveTo(r.X+2, r.Y+5)
							path.LineTo(r.X+6, r.Y+5)
							path.LineTo(r.X+6, r.Y+3)
							path.LineTo(r.X+10, r.Y+3)
							path.LineTo(r.X+10, r.Y+5)
							path.LineTo(r.X+14, r.Y+5)
							path.LineTo(r.X+14, r.Y+9)
							path.LineTo(r.X+12, r.Y+9)
							path.LineTo(r.X+12, r.Y+13)
							path.LineTo(r.X+8, r.Y+13)
							path.LineTo(r.X+8, r.Y+11)
							path.LineTo(r.X+4, r.Y+11)
							path.LineTo(r.X+4, r.Y+9)
							path.LineTo(r.X+2, r.Y+9)
							path.Close()
							p.StrokePath(path, 1.2, ink)
						})
					}, i18n.T(i18n.KeySidebarMarket), m.OnOpenMarket)
					nav(func() {
						ui.Box(c).Width(16).Height(16).Draw(func(p *ui.Painter, r ui.Rect) {
							circ := &ui.Path{}
							circ.Circle(r.X+8, r.Y+8, 6.5)
							p.StrokePath(circ, 1.2, muted)
							hand := &ui.Path{}
							hand.MoveTo(r.X+8, r.Y+4)
							hand.LineTo(r.X+8, r.Y+8)
							hand.LineTo(r.X+11, r.Y+10)
							p.StrokePath(hand, 1.2, muted)
						})
					}, i18n.T(i18n.KeySidebarSchedule), m.OnOpenSchedule)
					assist := ui.Button(c, "").FillWidth().Height(36).Radius(12).
						Background(ui.Transparent).BorderWidth(0).
						Padding(7, 8, 7, 8).AlignItems(ui.Center)
					assist.Children(func() {
						ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
							if bmp := chrome.BrandAvatar(); bmp != nil {
								ui.Image(c, bmp).Width(18).Height(18).Fit(ui.Contain)
							} else {
								ui.Text(c, "◉").FontSize(theme.InterfaceSize(14)).TextColor(muted)
							}
							ui.Text(c, i18n.T(i18n.KeySidebarAssistant)).FontSize(theme.InterfaceSize(14)).TextColor(fg).Grow(1)
						})
					})
					if m.OnOpenAssistant != nil && assist.Clicked() {
						m.OnOpenAssistant()
					}
				})
				menuOpen := &workspaceMenuFB
				if m.WorkspaceMenuOpen != nil {
					menuOpen = m.WorkspaceMenuOpen
				}
				// SidebarRoot.regionArea (flex:1): workspace browser + real session tree.
				ui.Column(c).FillWidth().Grow(1).MinHeight(0).Gap(6).Padding(8, 0, 0, 0).Children(func() {
					workspaceHeading(c, m, menuOpen, fg, muted)
					batchToolbar(c, m, fg)
					ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Gap(2).Children(func() {
						if sessionSearchResultsView(c, m, fg, muted) {
							projects.View(c, m.Projects)
							return
						}
						items := sidebarItems(m.Snapshot, m.SearchQuery)
						filter := browseArchiveFilter(m)
						if filter != "only" {
							workspaceRowsView(c, m, items, fg, muted, border)
						} else {
							workspaceRowsStatus(c, m, fg)
						}
						workspaceListFeedback(c, m, items, fg, muted)
						projects.View(c, m.Projects)
						if filter != "hide" && len(m.Snapshot.ArchivedSessionIDs) > 0 {
							ui.Text(c, i18n.T(i18n.KeySidebarArchivedSection)).FontSize(theme.InterfaceSize(11)).Bold().TextColor(muted).
								Padding(8, 0, 2, 0)
							for _, id := range m.Snapshot.ArchivedSessionIDs {
								aid := id
								label := i18n.T(i18n.KeySidebarNewSessionFallback)
								for _, it := range m.Snapshot.Items {
									if it.SessionID == aid && strings.TrimSpace(it.Title) != "" {
										label = it.Title
										break
									}
								}
								if batchRow(c, m, aid, label, muted) {
									continue
								}
								rowArchivedSession(c, aid, label, m, muted)
							}
						}
					})
				})
			}
			// Keep the same root/footer/settings identity across widths, as SidebarRoot does.
			// SidebarRoot.footArea: footer.action then settings (always pinned), and the
			// account seat at the very bottom.
			ui.Column(c.Key("sidebar-footer")).FillWidth().Padding(8, 0, 4, 0).Gap(2).Shrink(0).Children(func() {
				// The footer's top edge drags the window, like the logo row does.
				ui.Box(c).Key("sidebar-footer-drag").FillWidth().Height(6).DragWindow()
				// The launcher is not a sidebar row in the installed build: the header's
				// ellipsis opens it (conversation-more -> OnOpenLauncher), so the tree stays
				// to workspaces and sessions.

				// SidebarRoot.footArea: the account seat is the settings entry
				// once a profile is signed in; a signed-out profile keeps the
				// settings gear.
				if strings.TrimSpace(m.Account.Name) != "" || m.Account.SignedIn {
					accountRow(c, m, colors, muted, m.Collapsed)
				} else if m.OnOpenSettings != nil {
					settingsConnectionTrigger(c, m, colors, m.Collapsed)
				}
			})
		})
}

func rowSession(c *ui.Context, it session.Item, depth int, m Model, fg, muted, border ui.Color) {
	if batchRow(c, m, it.SessionID, it.Title, fg) {
		return
	}
	selected := it.SessionID == m.Snapshot.SelectedID
	renaming := m.RenameSessionID == it.SessionID && m.RenameDraft != nil
	status := sessionStatus(m, it)
	label := it.Title
	if it.Blank {
		label = i18n.T(i18n.KeySidebarNewSessionFallback)
	}
	if strings.TrimSpace(label) == "" {
		label = workspaceText("未命名会话", "Untitled session")
	}
	hint := label
	if it.Cwd != "" {
		hint += "\n" + it.Cwd
	}
	if text := statusLabel(status); text != "" {
		hint += "\n" + text
	}
	tasks := schedstore.ActiveForSession(m.Schedules, it.SessionID)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	row := ui.Button(c, "").Label(label).Description(hint).FillWidth().Height(32).Radius(12).
		Background(ui.Transparent).BorderWidth(0).Padding(0, 8, 0, float32(8+depth*12)).Clip().AlignItems(ui.Center)
	if selected || row.Hovered() || row.FocusWithin() {
		row.Background(selectedBackground(colors))
	}
	menu := sessionRowMenu(m, it)
	if !it.Blank && !renaming {
		row.ContextMenu(menu)
	}
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(0).Children(func() {
			if status == "" && len(tasks) > 0 {
				sessionScheduleMark(c, len(tasks), muted)
			} else {
				sessionStatusSlot(c, status, muted)
			}
			if renaming {
				input := ui.TextInput(c.Key("session-rename"), m.RenameDraft).MinWidth(0).Grow(1).Height(24).
					FontSize(theme.InterfaceSize(14)).Disabled(m.RenameBusy)
				first := ui.Local(input, "rename-focus", func() bool { return true })
				if *first {
					input.Focus()
					*first = false
				}
				if input.Changed() && m.OnRenameDraftChanged != nil {
					m.OnRenameDraftChanged(*m.RenameDraft)
				}
				if input.Submitted() && !m.RenameBusy && m.OnCommitRename != nil {
					m.OnCommitRename(it.SessionID, *m.RenameDraft)
				}
				if input.Shortcut(0, ui.KeyEscape) && !m.RenameBusy && m.OnCancelRename != nil {
					m.OnCancelRename()
				}
				if ui.Button(c, "×").Label(i18n.T(i18n.KeySidebarRenameCancel)).Width(16).Height(20).Padding(0).
					BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(m.RenameBusy).Clicked() && m.OnCancelRename != nil {
					m.OnCancelRename()
				}
				return
			}
			sessionTitleCell(c, row, it.SessionID, label, fg)
			if !it.Blank {
				showActions := row.Hovered() || row.FocusWithin()
				if !showActions {
					ui.Text(c, sessionTrailingLabel(status, it.UpdatedAt)).Label("Session time " + it.SessionID).
						FontSize(theme.InterfaceSize(10)).FixedLineHeight(16).TextColor(muted).SingleLine()
					if it.Pinned {
						sessionPinIndicator(c, it.SessionID, muted)
					}
				}
				// Rows.module.css: hover replaces the time cell with bare 16px
				// glyphs at a 10 DIP gap; the hidden group stays in the tree so
				// Tab still reaches the actions menu.
				actions := ui.Row(c).Key("session-actions-"+it.SessionID).Shrink(0).AlignItems(ui.Center).Gap(10)
				if !showActions {
					actions.Absolute().Right(4).Opacity(0)
				}
				actions.Children(func() {
					if showActions {
						sessionRowAction(c, "pin-"+it.SessionID, pinArtwork, sessionPinLabel(it.Pinned),
							sessionPinTip(it.Pinned), muted, pinAction(m, it))
						sessionRowAction(c, "archive-"+it.SessionID, archiveArtwork, i18n.T(i18n.KeySidebarArchive),
							i18n.T(i18n.KeySidebarArchive), muted, archiveAction(m, it))
					}
					more := ui.Button(c.Key("session-menu-"+it.SessionID), "…").Label(sessionActionsLabel(it.SessionID)).
						Tooltip(workspaceText("会话操作", "Session actions")).Width(16).Height(16).Padding(0).
						BorderWidth(0).Background(ui.Transparent).TextColor(muted).Menu(menu)
					_ = more
				})
			}
		})
	})
	if len(tasks) == 0 {
		sessionHoverCard(c, m, row, it, label, status, colors)
	} else {
		sessionScheduleHover(c, m, row, it, status, tasks, m.Schedules.ConnectionEpoch, colors)
	}
	if !renaming {
		sessionDragSource(row, m, it)
		sessionDropTarget(row, m, it, false, fg)
		if row.Clicked() && m.OnSelect != nil {
			m.OnSelect(it.SessionID)
		}
	}
	if m.sessionDrag != nil {
		after := ui.Box(c.Key(sessionRowKey{m.Snapshot.WorkspaceLayout.HostGeneration, m.sessionDrag.Workspace, "after/" + it.SessionID})).FillWidth().Height(5)
		sessionDropTarget(after, m, it, true, fg)
	}
	if renaming && m.RenameError != "" {
		ui.Text(c, m.RenameError).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	_ = border
}

// Selected rows use the theme's interactive fill and retain primary/tertiary
// label roles. This also keeps custom colors supplied by the caller intact.
func selectedBackground(colors theme.Colors) ui.Color {
	bg := chrome.Color(colors.Hover)
	if bg.A == 0 {
		bg = chrome.Color(colors.Field)
	}
	return bg
}

func buildTree(items []session.Item, pinnedOrder []string) []treeNode {
	byID := map[string]session.Item{}
	children := map[string][]session.Item{}
	var roots []session.Item
	for _, it := range items {
		byID[it.SessionID] = it
	}
	for _, it := range items {
		if _, parentPresent := byID[it.ParentID]; it.ParentID == "" || !parentPresent {
			roots = append(roots, it)
		} else {
			children[it.ParentID] = append(children[it.ParentID], it)
		}
	}
	session.SortItemsSidebar(roots, pinnedOrder)
	var build func(it session.Item) treeNode
	build = func(it session.Item) treeNode {
		kids := children[it.SessionID]
		session.SortItemsSidebar(kids, pinnedOrder)
		n := treeNode{item: it}
		for _, k := range kids {
			n.children = append(n.children, build(k))
		}
		return n
	}
	out := make([]treeNode, 0, len(roots))
	for _, r := range roots {
		out = append(out, build(r))
	}
	return out
}
