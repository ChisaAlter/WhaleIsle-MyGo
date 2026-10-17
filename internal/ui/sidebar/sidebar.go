// Package sidebar draws the workspace and session tree stub for M4.
package sidebar

import (
	"strings"

	"github.com/egoist/mygo/ui"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
)

// Model is sidebar state for one frame.
type Model struct {
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

	Projects         projects.Model
	HideArchivedList bool
	Collapsed        bool
	OnToggle         func()
	Snapshot         session.ViewState
	ThemeID          string
	SystemDark       bool
	SearchQuery      *string
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
	bg := chrome.BackdropSidebar(chrome.Color(colors.Sidebar))
	border := chrome.Color(colors.Border)
	if m.Collapsed {
		ui.Column(c).Fill().MinWidth(0).Padding(6, 8, 6, 8).Background(bg).Children(func() {
			if ui.Button(c, "›").Label(i18n.T(i18n.KeySidebarExpand)).Tooltip(i18n.T(i18n.KeySidebarExpand)+" (Ctrl+B)").FillWidth().Height(40).Clicked() && m.OnToggle != nil {
				m.OnToggle()
			}
		})
		return
	}
	// SidebarRoot.root: flex column height 100%, pad 6 / inline 12.
	// Children (flex:none): logoRow (60), newSession, panelList;
	// regionArea flex:1 min-height:0; footArea flex:none (pinned bottom).
	root := ui.Column(c).Fill().MinWidth(0).Padding(6, 12, 6, 12)
	if m.SearchArchivedFilter == nil {
		m.SearchArchivedFilter = ui.Local(root, struct {
			Key  string
			Host uint64
		}{"search-archive-filter", m.Snapshot.WorkspaceLayout.HostGeneration}, func() string { return "default" })
	}
	root.Background(bg).
		Children(func() {
			// SidebarRoot.module.css .logoRow height: 60px.
			ui.Row(c).FillWidth().Height(60).Gap(8).AlignItems(ui.Center).Children(func() {
				if bmp := chrome.BrandAvatar(); bmp != nil {
					ui.Image(c, bmp).Width(28).Height(28).Fit(ui.Contain)
				} else {
					ui.Box(c).Width(28).Height(28).Radius(14).Background(chrome.Color(colors.Accent).Alpha(0.25))
				}
				ui.Column(c).Gap(0).Grow(1).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, i18n.T(i18n.KeySidebarBrand)).FontSize(theme.InterfaceSize(15)).FontWeight(400).TextColor(fg)
						ui.Text(c, "WHALE ISLE").FontSize(theme.InterfaceSize(11)).FontWeight(400).TextColor(muted)
					})
					ui.Text(c, "BASED ON DEEPSEEK HARNESS").FontSize(theme.InterfaceSize(9)).TextColor(muted)
				})
				if m.OnToggle != nil {
					if ui.Button(c, "‹").Label(i18n.T(i18n.KeySidebarCollapse)).Tooltip(i18n.T(i18n.KeySidebarCollapse) + " (Ctrl+B)").Width(24).Height(24).BorderWidth(0).Background(ui.Transparent).Clicked() {
						m.OnToggle()
					}
				}
			})
			if m.OnCreate != nil {
				// SidebarRoot.module.css .newSession: h38, radius 12, margin 0 2 8, border-l2.
				btn := ui.Button(c, "+  "+i18n.T(i18n.KeySidebarNewSessionFallback)).FillWidth().Height(38).Radius(12).
					Background(chrome.Color(colors.Field)).Border(1, border).
					TextColor(fg).FontSize(theme.InterfaceSize(14)).FontWeight(500).Padding(8, 16, 8, 16)
				if btn.Clicked() {
					m.OnCreate()
				}
				ui.Box(c).Height(8)
			}
			// SidebarRoot.panelList gap 4; .panelRow min-h 36, pad 7 8, radius-md, gap 8.
			ui.Column(c).FillWidth().Gap(4).Padding(0, 0, 8, 0).Children(func() {
				nav := func(drawIcon func(), label string, fn func()) {
					row := ui.Button(c, "").FillWidth().Height(36).Radius(12).
						Background(ui.Transparent).BorderWidth(0).
						Padding(7, 8, 7, 8).AlignItems(ui.Center)
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
			label := m.WorkspaceLabel
			if label == "" {
				label = "Whale Isle"
			}
			base := label
			if i := strings.LastIndexAny(label, `/\`); i >= 0 && i+1 < len(label) {
				base = label[i+1:]
			}
			menuOpen := &workspaceMenuFB
			if m.WorkspaceMenuOpen != nil {
				menuOpen = m.WorkspaceMenuOpen
			}
			projects.View(c, m.Projects)
			// SidebarRoot.regionArea (flex:1): workspace browser + real session tree.
			ui.Column(c).FillWidth().Grow(1).MinHeight(0).Gap(6).Padding(8, 0, 0, 0).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, i18n.T(i18n.KeySidebarWorkspaceSection)).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(muted).Grow(1)
					if m.OnRefresh != nil {
						if ui.Button(c, "⌕").FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).
							TextColor(muted).Clicked() {
							m.OnRefresh()
						}
					} else {
						ui.Text(c, "⌕").FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
					if ui.Button(c, "☰").FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).
						TextColor(muted).Clicked() {
						*menuOpen = !*menuOpen
					}
					if m.OnCreate != nil {
						if ui.Button(c, "＋").FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).
							TextColor(muted).Clicked() {
							m.OnCreate()
						}
					} else {
						ui.Text(c, "＋").FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
				})
				if *menuOpen {
					menuBG := chrome.Color(colors.Field)
					ui.Column(c).FillWidth().Background(menuBG).Border(1, border).
						Radius(10).Padding(6, 8, 6, 8).Gap(2).Children(func() {
						workspaceViewOptions(c, m)
						if ui.Button(c, i18n.T(i18n.KeySidebarBrowseWorkspace)).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).
							TextColor(fg).FillWidth().Clicked() {
							*menuOpen = false
							if m.OnBrowseWorkspace != nil {
								m.OnBrowseWorkspace()
							}
						}
						if ui.Button(c, i18n.T(i18n.KeySidebarRevealWorkspace)).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).
							TextColor(fg).FillWidth().Clicked() {
							*menuOpen = false
							if m.OnRevealWorkspace != nil {
								m.OnRevealWorkspace()
							}
						}
						if m.OnRenameWorkspace != nil {
							if ui.Button(c, i18n.T(i18n.KeySidebarRenameWorkspace)).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).TextColor(fg).FillWidth().Clicked() {
								*menuOpen = false
								m.OnRenameWorkspace("")
							}
						}
						if m.OnDeleteWorkspace != nil {
							if ui.Button(c, i18n.T(i18n.KeySidebarDeleteWorkspace)).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).
								TextColor(fg).FillWidth().Clicked() {
								*menuOpen = false
								m.OnDeleteWorkspace()
							}
						}
					})
				}
				// Rows.module.css .projectRow height 34.
				folder := ui.Button(c, "").FillWidth().Height(34).BorderWidth(0).Background(ui.Transparent).
					Padding(0, 0, 0, 0).Radius(8)
				folder.Children(func() {
					ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "📁").FontSize(theme.InterfaceSize(13))
						ui.Text(c, base).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1)
					})
				})
				if folder.Clicked() && m.OnBrowseWorkspace != nil {
					m.OnBrowseWorkspace()
				}
				if m.SearchQuery != nil {
					searchInput := ui.TextInput(c, m.SearchQuery).FillWidth().Placeholder(i18n.T(i18n.KeySidebarSearch))
					if m.SearchFocus != nil && *m.SearchFocus {
						searchInput.Focus()
						*m.SearchFocus = false
					}
				}
				batchToolbar(c, m, fg)
				ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Gap(2).Children(func() {
					if sessionSearchResultsView(c, m, fg, muted) {
						return
					}
					items := sidebarItems(m.Snapshot, m.SearchQuery)
					workspaceRowsView(c, m, items, fg, muted, border)
					if !m.HideArchivedList && len(m.Snapshot.ArchivedSessionIDs) > 0 {
						ui.Text(c, i18n.T(i18n.KeySidebarArchivedSection)).FontSize(theme.InterfaceSize(11)).Bold().TextColor(muted).
							Padding(8, 0, 2, 0)
						for _, id := range m.Snapshot.ArchivedSessionIDs {
							aid := id
							label := aid
							for _, it := range m.Snapshot.Items {
								if it.SessionID == aid && strings.TrimSpace(it.Title) != "" {
									label = it.Title
									break
								}
							}
							if batchRow(c, m, aid, label, muted) {
								continue
							}
							ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Padding(4, 8, 4, 8).Children(func() {
								ui.Text(c, label).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1)
								if m.OnUnarchive != nil {
									if ui.Button(c, i18n.T(i18n.KeySidebarUnarchive)).FontSize(theme.InterfaceSize(11)).Clicked() {
										m.OnUnarchive(aid)
									}
								}
								if m.OnDelete != nil && ui.Button(c, i18n.T(i18n.KeySidebarDelete)).FontSize(theme.InterfaceSize(11)).Clicked() {
									m.OnDelete(aid)
								}
							})
						}
					}
				})
			})
			// SidebarRoot.footArea: footer.action then settings (always pinned).
			ui.Column(c).FillWidth().Padding(8, 0, 4, 0).Gap(2).Children(func() {
				if m.OnOpenLauncher != nil {
					btn := ui.Button(c, i18n.T(i18n.KeyLauncherOpen)).FillWidth().Height(32).Radius(8).
						BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(13)).
						Padding(0, 8, 0, 8).AlignItems(ui.Center)
					if btn.Clicked() {
						m.OnOpenLauncher()
					}
				}
				if m.OnContactAccount != nil && ui.Button(c, "联系与反馈 / Contact us").FillWidth().Height(32).Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
					m.OnContactAccount()
				}
				if m.OnOpenSettings != nil {
					btn := ui.Button(c, i18n.T(i18n.KeySettingsOpen)).FillWidth().Height(32).Radius(8).
						BorderWidth(0).Background(ui.Transparent).TextColor(fg).FontSize(theme.InterfaceSize(13)).
						Padding(0, 8, 0, 8).AlignItems(ui.Center)
					if btn.Clicked() {
						m.OnOpenSettings()
					}
				}
			})
		})
}

func rowSession(c *ui.Context, it session.Item, depth int, m Model, fg, muted, border ui.Color) {
	if batchRow(c, m, it.SessionID, it.Title, fg) {
		return
	}
	pad := float32(8 + depth*12)
	selected := it.SessionID == m.Snapshot.SelectedID
	renaming := m.RenameSessionID == it.SessionID && m.RenameDraft != nil
	label := it.Title
	if depth > 0 && !renaming {
		label = "↳ " + label
	}
	if it.Pinned && !renaming {
		label = i18n.T(i18n.KeySidebarPinnedPrefix) + label
	}
	status := sessionStatus(m, it)
	if !renaming {
		label = statusPrefix(status) + label
	}
	taskHint := ""
	for _, task := range schedstore.ActiveForSession(m.Schedules, it.SessionID) {
		taskHint += task.Title + " · " + schedstore.TimingLabel(task) + "\n"
	}
	if taskHint != "" && !it.Running && !renaming {
		label = "◷ " + label
	}
	titleColor := fg
	metaColor := muted
	bg := ui.Transparent
	if selected {
		bg = selectedBackground(theme.Resolve(m.ThemeID).Active(m.SystemDark))
	}
	// WorkspaceBrowser Rows.module.css: .sessionRow height 32; selected uses hover fill.
	if renaming {
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Padding(4, 8, 4, 8).Children(func() {
			inp := ui.TextInput(c, m.RenameDraft).Grow(1).Disabled(m.RenameBusy)
			if inp.Changed() && m.OnRenameDraftChanged != nil {
				m.OnRenameDraftChanged(*m.RenameDraft)
			}
			if inp.Submitted() && !m.RenameBusy && m.OnCommitRename != nil {
				m.OnCommitRename(it.SessionID, *m.RenameDraft)
			}
			if ui.Button(c, i18n.T(i18n.KeySidebarRenameCancel)).Disabled(m.RenameBusy).Clicked() && m.OnCancelRename != nil {
				m.OnCancelRename()
			}
		})
		if m.RenameError != "" {
			ui.Text(c, m.RenameError).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		return
	}
	if !selected {
		row := ui.Button(c, "").Tooltip(statusLabel(status)+"\n"+taskHint).FillWidth().Height(32).Radius(8).
			Background(bg).BorderWidth(0).
			Padding(0, 8, 0, pad).AlignItems(ui.Center)
		row.Children(func() {
			ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(titleColor).Grow(1)
		})
		sessionDragSource(row, m, it)
		sessionDropTarget(row, m, it, false, fg)
		if m.sessionDrag != nil {
			after := ui.Box(c.Key(sessionRowKey{m.Snapshot.WorkspaceLayout.HostGeneration, m.sessionDrag.Workspace, "after/" + it.SessionID})).FillWidth().Height(5)
			sessionDropTarget(after, m, it, true, fg)
		}
		if m.OnSelect != nil && row.Clicked() {
			m.OnSelect(it.SessionID)
		}
		_ = metaColor
		_ = border
		return
	}
	// Selected: title + optional cwd + action strip (overflow menu proxy).
	cell := ui.Column(c).Tooltip(statusLabel(status)).FillWidth().Padding(6, 8, 6, pad).Gap(4).Background(bg).Radius(8)
	cell.Children(func() {
		title := ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(titleColor)
		sessionDragSource(title, m, it)
		if taskHint != "" {
			ui.Text(c, strings.TrimSpace(taskHint)).FontSize(theme.InterfaceSize(11)).TextColor(metaColor)
		}
		if it.Cwd != "" {
			ui.Text(c, it.Cwd).FontSize(theme.InterfaceSize(11)).TextColor(metaColor)
		}
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			if it.Pinned {
				if m.OnUnpin != nil && ui.Button(c, i18n.T(i18n.KeySidebarUnpin)).Tooltip(i18n.T(i18n.KeySidebarUnpin)).FontSize(theme.InterfaceSize(11)).Clicked() {
					m.OnUnpin(it.SessionID)
				}
			} else if m.OnPin != nil && ui.Button(c, i18n.T(i18n.KeySidebarPin)).Tooltip(i18n.T(i18n.KeySidebarPin)).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.OnPin(it.SessionID)
			}
			if m.OnMoveSessionBefore != nil && ui.Button(c, i18n.T(i18n.KeySidebarMoveUp)).Tooltip(i18n.T(i18n.KeySidebarMoveUp)).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.OnMoveSessionBefore(it.SessionID, "")
			}
			if m.OnBeginRename != nil && ui.Button(c, i18n.T(i18n.KeySidebarRename)).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.OnBeginRename(it.SessionID, it.Title)
			}
			if m.OnArchive != nil && ui.Button(c, i18n.T(i18n.KeySidebarArchive)).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.OnArchive(it.SessionID)
			}
			if m.OnCopySessionID != nil && ui.Button(c, i18n.T(i18n.KeySidebarCopyID)).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.OnCopySessionID(it.SessionID)
			}
			if m.OnFork != nil && ui.Button(c, i18n.T(i18n.KeySidebarFork)).FontSize(theme.InterfaceSize(11)).Clicked() {
				m.OnFork(it.SessionID)
			}
		})
	})
	sessionDropTarget(cell, m, it, false, fg)
	if m.sessionDrag != nil {
		after := ui.Box(c.Key(sessionRowKey{m.Snapshot.WorkspaceLayout.HostGeneration, m.sessionDrag.Workspace, "after/" + it.SessionID})).FillWidth().Height(5)
		sessionDropTarget(after, m, it, true, fg)
	}
	if m.OnSelect != nil && cell.Clicked() {
		m.OnSelect(it.SessionID)
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
