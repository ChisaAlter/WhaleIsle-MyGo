// Package sidebar draws the workspace and session tree: search, create, pin,
// archive, delete and rename over the session store.
package sidebar

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/product"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/workspacepicker"
)

// whaleAssistantNavLabel renders only the published snapshot. The name is
// fetched through the Whale settings observation flow (Host plugin RPC); this
// draw path never touches disk.
func whaleAssistantNavLabel(m Model) string {
	fallback := i18n.T(i18n.KeySidebarAssistant)
	if !m.ConnectionReady {
		return fallback
	}
	if name := strings.TrimSpace(m.AssistantName); name != "" {
		return name
	}
	return fallback
}

func sidebarUpdatePhase(m Model) string {
	if m.DesktopUpdateInstalling {
		return "installing"
	}
	switch m.DesktopUpdatePhase {
	case "checking", "available", "downloading", "verifying", "installing", "ready":
		return m.DesktopUpdatePhase
	}
	return ""
}

// viewport is the window's client area in DIPs: the frame's measurement when the
// sidebar is painted inside it, Context.Size otherwise (tests, previews).
func (m Model) viewport(c *ui.Context) (width, height float32) {
	if m.ViewportW > 0 && m.ViewportH > 0 {
		return m.ViewportW, m.ViewportH
	}
	return c.Size()
}

var sidebarVirtualRowHook func(surface string, index int, id string)

func sidebarListMaxHeight(m Model, c *ui.Context, ratio float32) float32 {
	_, winH := m.viewport(c)
	if winH <= 0 {
		winH = 600
	}
	if ratio >= 0.99 {
		return max(120, winH)
	}
	return max(120, winH*ratio)
}

// PluginEntry represents one custom installed plugin row in the left sidebar.
type PluginEntry struct {
	ID    string
	Label string
	Order int
}

// Model is sidebar state for one frame.
type Model struct {
	pinnedSection      bool
	scrollbarsVisible  bool
	sidebarBodyFill    bool
	sessionContext     string
	schedulesBySession map[string][]schedstore.Entry
	DirectoryActions   []workspacepicker.DirectoryAction
	// ViewportW/H are the window's client area in DIPs. Context.Size reports
	// device pixels on a scaled display, so the hover cards take their bounds
	// from here and fall back to Context.Size only when the frame did not say.
	ViewportW       float32
	ViewportH       float32
	ConnectionReady bool
	// AssistantName is the published Whale display name snapshot; empty falls
	// back to the localized assistant label. Populated by the Host-side
	// settings observer, never read from disk here.
	AssistantName           string
	ConnectionError         string
	DesktopUpdateInstalling bool
	DesktopUpdatePhase      string
	DesktopUpdateDetail     string
	OnOpenDesktopUpdate     func()
	ConnectionConnecting    bool
	// HostNote is what the empty session list says while the local service is
	// not ready (internal/hoststate). It outranks the workspace copy: with no
	// service there are no sessions to connect to yet.
	HostNote        string
	ConnectionEpoch uint64
	OnReconnect     func()
	PendingStatuses map[string]string
	sessionDrag     *sessionDragContext
	sessionHover    *sessionHoverState
	Batch           session.SessionBatch
	OnBatchEdit     func(string, []session.SessionBatchTarget)
	OnBatchRequest  func(string)
	OnBatchConfirm  func(uint64)
	OnBatchCancel   func(uint64)
	// ArchiveUndo is the store's live undo offer after an archive; the sidebar
	// renders it as a receipt row with a real undo button until it expires.
	ArchiveUndo                 *session.ArchiveUndoNotice
	OnUndoArchive               func(uint64)
	Schedules                   schedstore.ViewState
	OnRetryWorkspacePreferences func()
	OnDropWorkspaceBefore       func(uint64, string, string)
	OnSelectWorkspace           func(string)
	// OnCreateWorkspaceSession opens a session in the addressed Host workspace.
	OnCreateWorkspaceSession     func(uint64, string)
	OnManageWorkspace            func(string)
	OnBeginWorkspaceRename       func(string)
	OnBeginWorkspaceRemoval      func(string)
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
	// OnArchivedFilter writes the shared Host-scoped preference. When present,
	// Snapshot.WorkspaceLayout.ArchivedFilter is the sole live value.
	OnArchivedFilter func(string)
	// BrowseArchivedFilter is a compatibility binding for standalone callers.
	// Browsing and search both read this same choice; product uses the callback.
	BrowseArchivedFilter *string
	archiveExpanded      *bool
	searchState          *sessionSearchState
	workspaceWatermarks  *workspaceWatermarkState
	sidebarDrag          *sidebarDragState
	Collapsed            bool
	OnToggle             func()
	Snapshot             session.ViewState
	ThemeID              string
	SystemDark           bool
	SearchQuery          *string
	// SearchArchivedFilter is a compatibility alias of BrowseArchivedFilter.
	SearchArchivedFilter *string
	SearchFocus          *bool
	WorkspaceLabel       string
	OnSelect             func(sessionID string)
	OnRefresh            func()
	OnCreate             func()
	// ActivePanel is the built-in route or live NativeView handle; empty means conversation.
	ActivePanel           string
	NewSessionShortcut    string
	MenuShortcutLabels    map[string]string
	OnNoDirectory         func()
	OnOpenMarket          func()
	PluginEntries         []PluginEntry
	OnOpenPlugin          func(string)
	OnCreatePlugin        func()
	OnOpenSchedule        func()
	OnOpenAssistant       func()
	OnOpenDesktopSettings func()
	OnContactAccount      func()
	OnOpenSettings        func()
	// OnBrowseWorkspace opens the folder picker (SidebarRoot ☰ / workspace row).
	OnBrowseWorkspace func()
	// OnRevealWorkspace opens the current workspace in the OS file manager.
	OnRevealWorkspace     func()
	OnRevealWorkspacePath func(id, path string)
	ArchiveConfirm        *ArchiveConfirmState
	DeleteConfirm         *DeleteConfirmState
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

// sidebarBrand keeps the caption drag surface outside the non-macOS session button.
func sidebarBrand(c *ui.Context, m Model, platform string) {
	t := tok(c)
	fg, muted := t.Foreground, t.MutedForeground
	skinLayout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark)
	identity := func() {
		if skinLayout.Skin == theme.SkinCodex {
			ui.Text(c, product.Name).Font(theme.RenderPreferences().Interface.Family).FontSize(15).FontWeight(design.WeightSemibold).LetterSpacing(-.15).Grow(1).MinWidth(0).SingleLine().TextColor(fg)
			return
		}
		if bmp := chrome.BrandAvatar(); bmp != nil {
			ui.Image(c, bmp).Size(38, 38).Fit(ui.Contain)
		} else {
			ui.Box(c).Size(38, 38).Radius(design.RadiusFull).Background(t.Primary.Alpha(0.25))
		}
		// Claude's brand scope uses the user's sans stack rather than Inter.
		ui.Column(c).Gap(design.Sp(0.25)).Grow(1).MinWidth(0).Font(theme.RenderPreferences().Interface.Family).Children(func() {
			ui.Row(c).Gap(design.Sp(1)).AlignItems(ui.Center).MinWidth(0).Children(func() {
				ui.Row(c).Label(product.Name).Gap(0).Shrink(0).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "鲸").FontSize(fs(14)).FixedLineHeight(fs(20)).
						FontWeight(design.WeightBold).TextColor(fg).SingleLine()
					ui.Text(c, "屿 MyGo").FontSize(fs(14)).FixedLineHeight(fs(20)).
						FontWeight(design.WeightBold).TextColor(t.Primary).SingleLine()
				})
				ui.Text(c, "WHALE ISLE MYGO").Shrink(0).FontSize(fs(8.5)).FixedLineHeight(fs(14)).
					FontWeight(design.WeightBold).LetterSpacing(0.1).TextColor(fg.Alpha(0.88)).SingleLine()
			})
			ui.Text(c, "BASED ON DEEPSEEK HARNESS").FontSize(fs(8.5)).
				FixedLineHeight(fs(11)).FontWeight(design.WeightMedium).LetterSpacing(0.28).TextColor(muted)
		})
	}
	if platform == "darwin" || m.OnCreate == nil {
		ui.Row(c.Key("sidebar-brand-identity")).Label(product.Name).Grow(1).MinWidth(0).Gap(design.Sp(1.75)).AlignItems(ui.Center).DragWindow().Children(identity)
		return
	}
	brand := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonMD,
		Label:   product.Name,
		Key:     "sidebar-brand-new",
		Content: func() {},
	}).Description(i18n.T(i18n.KeySidebarNewSessionFallback)).
		Grow(1).Shrink(1).MinWidth(0).Height(38).Gap(design.Sp(1.75)).Padding(0).BorderWidth(0).
		AlignItems(ui.Center).Justify(ui.Start).Radius(design.RadiusMD)
	brand.Children(identity)
	if brand.Clicked() {
		m.OnCreate()
	}
}

var newSessionIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3"><path d="M2.37091 11.2501C1.58745 9.89288 1.32067 8.29835 1.61969 6.76006C1.91872 5.22177 2.76342 3.8433 3.99826 2.87846C5.2331 1.91362 6.77494 1.42737 8.33988 1.50925C9.90482 1.59113 11.3875 2.23562 12.5149 3.32406C13.6425 4.41269 14.3387 5.87206 14.4754 7.4334C14.612 8.99474 14.18 10.5529 13.2587 11.8209C12.3375 13.0888 10.9891 13.9813 9.46194 14.3337C8.18691 14.628 6.85895 14.5294 5.64989 14.0605C5.1712 13.8748 4.76962 13.4932 4.26534 13.3967C3.67413 13.2835 2.95257 13.5598 2.03794 14.3337"/><path d="M8 5V11M5 8H11"/></svg>`))

// The title stays centered over the full button while its right edge fades
// before the configured shortcut, matching SidebarRoot's label mask.
func newSessionButton(c *ui.Context, m Model) {
	t := tok(c)
	fg := t.Foreground
	label := i18n.T(i18n.KeySidebarNewSessionFallback)
	if theme.ResolvedSkin(m.ThemeID, m.SystemDark) == theme.SkinCodex {
		btn := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonMD,
			Label:   label,
			Key:     "sidebar-new-session",
			Content: func() {},
		}).FillWidth().Height(32).Radius(design.RadiusMD).Margin(design.Sp(0.5), 0, design.Sp(1.5), 0).
			Padding(0, design.Sp(2)).BorderWidth(0).Justify(ui.Start).Gap(design.Sp(2)).TextColor(fg)
		btn.Children(func() {
			ui.Icon(c, newSessionIcon).Size(design.IconSM, design.IconSM).TextColor(fg)
			ui.Text(c, label).FontSize(13).FontWeight(design.WeightMedium).TextColor(fg)
		})
		if btn.Clicked() {
			m.OnCreate()
		}
		return
	}
	btn := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonOutline,
		Size:    coss.ButtonLG,
		Label:   label,
		Key:     "sidebar-new-session",
		Content: func() {},
	}).FillWidth().Height(38).Radius(design.RadiusXL).
		Margin(design.Sp(1), design.Sp(0.5), 0, design.Sp(0.5)).MaxWidth(252).
		TextColor(fg).FontSize(fs(14)).FontWeight(design.WeightMedium).
		Padding(design.Sp(2), design.Sp(3), design.Sp(2), design.Sp(3))
	showShortcut := m.NewSessionShortcut != "" && (btn.Hovered() || btn.FocusVisible())
	font := ui.Font{Family: c.Theme().Font, Size: fs(14), Weight: design.WeightMedium}
	type newSessionLayoutCache struct {
		label, shortcut, family string
		size                    float32
		title                   *ui.TextLayout
		titleW, titleH          float32
		shortcutW               float32
	}
	cache := ui.Local(btn, "sidebar-new-session-layouts", func() newSessionLayoutCache { return newSessionLayoutCache{} })
	if cache.title == nil || cache.label != label || cache.family != font.Family || cache.size != font.Size {
		cache.title = ui.ShapeTextLine(label, font, 0)
		cache.titleW, cache.titleH = cache.title.Size()
		cache.label, cache.family, cache.size = label, font.Family, font.Size
		cache.shortcut = ""
	}
	title, titleW, titleH := cache.title, cache.titleW, cache.titleH
	shortcutW := float32(0)
	if showShortcut {
		if cache.shortcut != m.NewSessionShortcut {
			cache.shortcut = m.NewSessionShortcut
			cache.shortcutW, _ = ui.ShapeTextLine(m.NewSessionShortcut, ui.Font{Family: font.Family, Size: font.Size, Weight: design.WeightNormal}, 0).Size()
		}
		shortcutW = cache.shortcutW
	}
	fadeBg := t.Popover
	if t.IsDark() {
		fadeBg = t.Input.Alpha(0.64).Over(t.Sidebar)
	} else if btn.Hovered() || btn.Pressed() {
		fadeBg = t.Accent.Alpha(0.5).Over(t.Popover)
	}
	btn.Children(func() {
		mask := ui.Box(c.Key("sidebar-new-label")).Grow(1).MinWidth(0).Height(22).PassThrough()
		mask.Draw(func(p *ui.Painter, r ui.Rect) {
			x := r.X + (r.W+shortcutW-titleW-20)/2
			p.Clip(r, 0, func() {
				p.Icon(newSessionIcon, ui.Rect{X: x, Y: r.Y + (r.H-14)/2, W: 14, H: 14}, fg)
				p.TextLayout(title, x+20, r.Y+(r.H-titleH)/2, fg)
				if showShortcut {
					fade := min(design.Sp(4), r.W)
					if fade > 0 {
						p.FillGradient(
							ui.Rect{X: r.X + r.W - fade, Y: r.Y, W: fade, H: r.H},
							ui.LinearGradient{From: fadeBg.Alpha(0), To: fadeBg, Angle: 90},
							0,
						)
					}
				}
			})
		})
		if showShortcut {
			ui.Text(c, m.NewSessionShortcut).Role(ui.RoleNone).Width(shortcutW).Shrink(1).MinWidth(0).SingleLine().
				FontSize(font.Size).FontWeight(design.WeightNormal).FixedLineHeight(22).TextColor(fg).PassThrough().
				ObserveLayout(func(r ui.Rect) { shortcutW = r.W })
		}
	})
	if btn.Clicked() {
		m.OnCreate()
	}
}

type treeNode struct {
	item     session.Item
	children []treeNode
}

// View paints the left session tree.
func View(c *ui.Context, m Model) {
	defer withSidebarTheme(c, m)()
	t := tok(c)
	fg := t.Foreground
	muted := t.MutedForeground
	border := t.SidebarBorder
	// SidebarRoot.root: flex column height 100%, pad 6 / inline 12.
	// Children (flex:none): logoRow (60), newSession, panelList;
	// regionArea flex:1 min-height:0; footArea flex:none (pinned bottom).
	skinLayout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark)
	root := ui.Column(c.Key("sidebar-root")).Fill().MinWidth(0).Padding(design.Sp(1.5), skinLayout.SidebarInline, design.Sp(1.5), skinLayout.SidebarInline)
	m = beginSessionHover(root, m)
	m.schedulesBySession = buildActiveScheduleMap(m.Schedules)
	if skinLayout.Active() {
		root.FontSize(skinLayout.SidebarFont)
	}
	if m.Collapsed {
		root.Padding(design.Sp(4.5), design.Sp(2.5), design.Sp(1.5), design.Sp(2.5))
	}
	if m.SearchFocus == nil {
		m.SearchFocus = ui.Local(root, "search-focus-request", func() bool { return false })
	}
	// The archive filter belongs to the Host-scoped store, not this retained UI.
	// Only the independent management section's expansion is session-local.
	m.archiveExpanded = ui.Local(root, "archive-expanded", func() bool { return false })
	m.searchState = ui.Local(root, "session-search", func() sessionSearchState {
		return sessionSearchState{Expanded: m.SearchQuery != nil && strings.TrimSpace(*m.SearchQuery) != ""}
	})
	m.workspaceWatermarks = ui.Local(root, "workspace-watermarks", func() workspaceWatermarkState { return workspaceWatermarkState{} })
	m.sidebarDrag = ui.Local(root, "sidebar-drag-state", func() sidebarDragState { return sidebarDragState{} })
	fill := t.Sidebar
	prefs := chrome.BackdropPreferences()
	blurSidebar := chrome.BackdropActive() && skinLayout.Skin != theme.SkinCodex &&
		!prefs.HideSidebarMask && !(prefs.Transparent && chrome.BackdropHasWallpaper())
	if skinLayout.Active() {
		if m.Collapsed && skinLayout.Skin == theme.SkinCodex {
			fill = t.Background
		}
		if skinLayout.Skin != theme.SkinCodex && chrome.BackdropActive() {
			chrome.GlassSurface(root, fill, chrome.MaterialSidebar).Radius(0)
			if blurSidebar {
				root.Material(glass.Blur{Radius: 16, Saturation: 0.35, Dither: true})
			}
		} else {
			root.Background(fill).Radius(0)
		}
		root.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X + r.W - skinLayout.SidebarBorder, Y: r.Y, W: skinLayout.SidebarBorder, H: r.H}, t.SidebarBorder, 0)
		})
	} else if chrome.BackdropActive() {
		chrome.GlassSurface(root, fill, chrome.MaterialSidebar).Radius(0)
		if blurSidebar {
			root.Material(glass.Blur{Radius: 16, Saturation: 0.35, Dither: true})
		}
		root.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X + r.W - design.BorderWidth, Y: r.Y, W: design.BorderWidth, H: r.H}, border, 0)
		})
	} else {
		root.Background(fill).Radius(0)
		if t.Glass || chrome.BackdropActive() {
			root.DrawOver(func(p *ui.Painter, r ui.Rect) {
				p.Fill(ui.Rect{X: r.X + r.W - design.BorderWidth, Y: r.Y, W: design.BorderWidth, H: r.H}, border, 0)
			})
		}
	}
	root.Children(func() {
		if m.Collapsed {
			rail := ui.Column(c.Key("sidebar-rail-content")).FillWidth().Grow(1).MinHeight(0)
			if !c.Preferences().ReduceMotion {
				rail.Transition(ui.ElementTransition{
					Duration: design.DurationDefault,
					Ease:     design.EaseInOut,
					Position: true,
					Enter:    &ui.Motion{X: 49, Opacity: 0},
				})
			}
			rail.Children(func() { collapsedRail(c, m, fg) })
		} else {
			wide := ui.Column(c.Key("sidebar-wide-content")).FillWidth().Grow(1).MinHeight(0)
			if !c.Preferences().ReduceMotion {
				wide.Transition(ui.ElementTransition{
					Duration: design.MotionDefault,
					Ease:     design.EaseInOut,
					Enter:    &ui.Motion{Opacity: 0},
				})
			}
			wide.Children(func() {
				ui.Row(c).Key("sidebar-brand-row").FillWidth().Height(skinLayout.BrandHeight).Margin(0, 0, skinLayout.BrandBottom, 0).Gap(design.Sp(1.5)).AlignItems(ui.Center).
					DragWindow().Children(func() {
					sidebarBrand(c, m, runtime.GOOS)
					if m.OnToggle != nil {
						collapse := coss.Button(c, "", coss.ButtonOptions{
							Variant: coss.ButtonGhost,
							Size:    coss.ButtonIconSM,
							Label:   i18n.T(i18n.KeySidebarCollapse),
							Key:     "sidebar-collapse",
							Content: func() {},
						}).Label(i18n.T(i18n.KeySidebarCollapse)).Tooltip(i18n.T(i18n.KeySidebarCollapse)+" (Ctrl+B)").
							Size(28, 28).Shrink(0).Padding(0).Radius(design.RadiusMD).BorderWidth(0).Center()
						collapseInk := t.MutedForeground
						if collapse.Hovered() {
							collapseInk = fg
						}
						collapse.Children(func() {
							ui.Icon(c, sidebarRailIcons["panel"]).Size(design.IconSM, design.IconSM).TextColor(collapseInk)
						})
						if collapse.Clicked() {
							m.OnToggle()
						}
					}
				})
				if m.OnCreate != nil {
					newSessionButton(c, m)
					if skinLayout.Skin != theme.SkinCodex {
						ui.Box(c).Height(12)
					}
				}
				ui.Column(c).Label(sidebarLocaleText("panels.label")).FillWidth().Padding(0, design.Sp(0.5), design.Sp(2), design.Sp(0.5)).Gap(design.Sp(0.5)).Children(func() {
					nav := func(id, key string, drawIcon func(), label string, fn func(), contextMenu func(*ui.Menu)) {
						row := coss.Button(c, "", coss.ButtonOptions{
							Variant: coss.ButtonGhost,
							Size:    coss.ButtonMD,
							Pressed: m.ActivePanel == id,
							Key:     "sidebar-panel-" + key,
							Content: func() {},
						}).FillWidth().Height(skinLayout.NavHeight).Radius(design.RadiusXL).
							TextColor(fg).BorderWidth(0).
							Padding(design.Sp(1.75), design.Sp(2), design.Sp(1.75), design.Sp(2)).AlignItems(ui.Center)
						active := m.ActivePanel == id
						if active {
							row.Background(sbSelectedBg(t))
							sbActiveIndicator(row, t)
						} else if row.Hovered() || row.FocusVisible() {
							row.Background(sbHoverBg(t))
						} else {
							row.Background(ui.Transparent)
						}
						if skinLayout.Skin == theme.SkinCodex {
							row.Padding(design.Sp(1.5), design.Sp(2)).MinHeight(32)
						}
						row.Children(func() {
							ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
								if drawIcon != nil {
									drawIcon()
								}
								lineHeight := fs(22)
								if skinLayout.Active() {
									lineHeight = 20
								}
								ui.Text(c, label).FontSize(skinLayout.SidebarFont).
									FixedLineHeight(lineHeight).TextColor(fg).Grow(1)
							})
						})
						if contextMenu != nil {
							row.ContextMenu(contextMenu)
						}
						if fn != nil && row.Clicked() {
							fn()
						}
					}
					for _, panel := range sidebarPanelRows(m) {
						panel := panel
						var menu func(*ui.Menu)
						if panel.id == "plugins" && m.OnCreatePlugin != nil {
							menu = func(ctxMenu *ui.Menu) {
								ctxMenu.CloseOnPointerLeave(true)
								if m.OnOpenMarket != nil && ctxMenu.Item(sidebarLocaleText("panels.browse")).Chosen() {
									m.OnOpenMarket()
								}
								if ctxMenu.Item(sidebarLocaleText("panels.create")).Chosen() {
									m.OnCreatePlugin()
								}
							}
						}
						nav(panel.active, "builtin:"+panel.id, func() { sidebarPanelIcon(c, panel.id, fg) }, panel.label, panel.run, menu)
					}
				})
				menuOpen := &workspaceMenuFB
				if m.WorkspaceMenuOpen != nil {
					menuOpen = m.WorkspaceMenuOpen
				}
				ui.Column(c).FillWidth().Grow(1).MinHeight(0).Gap(design.Sp(1)).Padding(design.Sp(0.6675), 0, 0, 0).Children(func() {
					sectionInk := t.SidebarForeground
					workspaceHeading(c, m, menuOpen, fg, sectionInk)
					batchToolbar(c, m, fg)
					m.scrollbarsVisible = sidebarScrollbarLinger(c, root)
					searchActive := m.SearchQuery != nil && session.NormalizeSessionSearchQuery(*m.SearchQuery) != ""
					if searchActive {
						m.sidebarBodyFill = true
						ui.Column(c.Key("sidebar-search-body")).FillWidth().Grow(1).MinHeight(0).Gap(design.Sp(0.5)).Children(func() {
							sessionSearchResultsView(c, m, fg, muted)
							// Project 功能已改版，侧栏区块暂时下线；恢复时取消注释。
							// projects.View(c, m.Projects)
						})
					} else {
						filter := browseArchiveFilter(m)
						items := sidebarItemsForArchiveFilter(m.Snapshot, m.SearchQuery, filter)
						archivedItems := archivedSectionItems(m)
						showArchived := (filter == "default" && !m.HideArchivedList) || len(archivedItems) > 0
						m.sidebarBodyFill = true
						ui.Column(c.Key("sidebar-tree-body")).FillWidth().Grow(1).MinHeight(0).Gap(design.Sp(0.5)).Children(func() {
							workspaceRowsView(c, m, items, fg, muted, border)
							workspaceListFeedback(c, m, items, fg, muted)
							// Project 功能已改版，侧栏区块暂时下线；恢复时取消注释。
							// projects.View(c, m.Projects)
							if showArchived {
								archivedSection(c, m, archivedItems, sectionInk, muted)
							}
						})
					}
				})
			})
		}
		batchResultReceipt(c, m)
		archiveUndoRow(c, m)
		ui.Column(c.Key("sidebar-footer")).FillWidth().Padding(design.Sp(2), 0, design.Sp(2), 0).Gap(design.Sp(0.5)).Shrink(0).Children(func() {
			// The footer's top edge drags the window, like the logo row does.
			ui.Box(c).Key("sidebar-footer-drag").FillWidth().Height(6).DragWindow()
			accountRow(c, m, m.Collapsed)
			updatePhase := sidebarUpdatePhase(m)
			if !m.Collapsed && updatePhase != "" {
				bottomOffset := design.Sp(1)
				if !accountSignedOut(m.Account) {
					bottomOffset = design.Sp(2)
				}
				ui.Box(c).Key("sidebar-footer-update-seat").Size(36, 36).Absolute().Right(design.Sp(0.5)).Bottom(bottomOffset).Children(func() {
					desktopUpdateSeat(c, m, updatePhase)
				})
			}
		})
	})
	renderSessionHover(c, m, root, m.sessionHover.dragging)
	sessionRenameDialog(c, m)
	sessionArchiveConfirmDialog(c, m)
	sessionDeleteConfirmDialog(c, m)
	batchDialog(c, m)
}

func rowSession(c *ui.Context, it session.Item, depth int, m Model, fg, muted, border ui.Color) {
	label := it.Title
	if it.Blank {
		label = i18n.T(i18n.KeySidebarNewSessionFallback)
	}
	if strings.TrimSpace(label) == "" {
		label = workspaceText("未命名会话", "Untitled session")
	}
	label = sessionRowLabel(label)
	if batchSessionRow(c, m, it, depth, label, fg, muted) {
		return
	}
	t := tok(c)
	selected := it.SessionID == displayedSessionID(m)
	renaming := m.RenameSessionID == it.SessionID && m.RenameDraft != nil
	status := sessionStatus(m, it)
	hint := label
	if it.Cwd != "" {
		hint += "\n" + it.Cwd
	}
	if text := statusLabel(status); text != "" {
		hint += "\n" + text
	}
	tasks := activeSchedulesForSession(m, it.SessionID)
	row := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonSM,
		Pressed: selected,
		Label:   label,
		Content: func() {},
	}).Description(hint).FillWidth().Height(32).Radius(design.RadiusMD).
		BorderWidth(0).Padding(0, design.Sp(2), 0, design.Sp(2)+float32(depth)*design.Sp(3)).Clip().AlignItems(ui.Center)
	revealSessionSearchRow(row, m, it.SessionID)
	if selected {
		row.Background(sbSelectedBg(t))
		sbActiveIndicator(row, t)
	} else if row.Hovered() || row.FocusWithin() || row.MenuOpen() {
		row.Background(sbHoverBg(t))
	} else {
		row.Background(ui.Transparent)
	}
	menu := sessionRowMenu(m, it)
	if !it.Blank && !renaming {
		row.ContextMenu(menu)
	}
	hoverPressed := row.Pressed()
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(0).Children(func() {
			if status == "" && !it.Blank {
				if len(tasks) > 0 {
					sessionScheduleMark(c, len(tasks), muted)
				} else {
					sessionStatusSlot(c, status, muted)
				}
			} else {
				sessionStatusSlot(c, status, muted)
			}
			title := sessionTitleCell(c, row, it.SessionID, label, fg)
			if m.sessionHover != nil {
				m.sessionHover.dragging = m.sessionHover.dragging || title.Dragging()
			}
			if !it.Blank && !renaming && m.OnBeginRename != nil {
				// Pressed makes its receiver a click owner. Only query the title
				// when its rename/select handlers actually own title input.
				hoverPressed = hoverPressed || title.Pressed()
				sessionDragSource(title, m, it)
				if title.DoubleClicked() {
					m.OnBeginRename(it.SessionID, it.Title)
				} else if title.Clicked() && m.OnSelect != nil {
					m.OnSelect(it.SessionID)
				}
			}
			if !it.Blank {
				showActions := row.Hovered() || row.FocusWithin() || row.MenuOpen()
				if !showActions {
					if m.pinnedSection {
						ui.Text(c, m.sessionContext).Label("Workspace " + it.SessionID).MaxWidth(64).MinWidth(0).
							FontSize(fs(10)).FixedLineHeight(16).TextColor(muted).SingleLine().Tooltip(m.sessionContext)
					} else {
						ui.Text(c, sessionTrailingLabel(status, it.UpdatedAt)).Label("Session time " + it.SessionID).
							FontSize(fs(10)).FixedLineHeight(16).TextColor(muted).SingleLine()
						if it.Pinned {
							sessionPinIndicator(c, it.SessionID, muted)
						}
					}
				}
				actions := ui.Row(c).Key("session-actions-" + it.SessionID).Shrink(0).AlignItems(ui.Center).Gap(design.Sp(3.75))
				if !showActions {
					actions.Absolute().Right(design.Sp(1)).Opacity(0)
				}
				actions.Children(func() {
					more := coss.Button(c, "", coss.ButtonOptions{
						Variant: coss.ButtonGhost,
						Size:    coss.ButtonIconXS,
						Label:   sessionActionsLabel(it.SessionID, label),
						Key:     "session-menu-" + it.SessionID,
						Content: func() {},
					}).Tooltip(workspaceText("会话操作", "Session actions")).Size(design.IconSM, design.IconSM).Padding(0).
						BorderWidth(0).TextColor(muted).Center().Menu(menu)
					more.Children(func() { ui.Icon(c, menuEllipsisArtwork).Size(design.IconSM, design.IconSM).TextColor(muted) })
					if showActions {
						sessionRowAction(c, "archive-"+it.SessionID, menuArchiveArtwork, i18n.T(i18n.KeySidebarArchive),
							i18n.T(i18n.KeySidebarArchive), muted, archiveAction(m, it))
						pinArt := menuPinArtwork
						if it.Pinned {
							pinArt = sessionPinArtwork
						}
						sessionRowAction(c, "pin-"+it.SessionID, pinArt, sessionPinLabel(it.Pinned),
							sessionPinTip(it.Pinned), muted, pinAction(m, it))
					}
				})
			}
		})
	})
	if m.sessionHover != nil {
		m.sessionHover.dragging = m.sessionHover.dragging || row.Dragging()
	}
	sessionHoverCard(c, m, row, it, label, status, hoverPressed)
	if !renaming {
		sessionDragSource(row, m, it)
		sessionRowDropTarget(row, m, it, fg)
		if row.Clicked() && m.OnSelect != nil {
			m.OnSelect(it.SessionID)
		}
	}
	_ = border
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
