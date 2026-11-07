// Package sidebar draws the workspace and session tree: search, create, pin,
// archive, delete and rename over the session store.
package sidebar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/workspacepicker"
)

var whaleSettingsCache struct {
	mu        sync.Mutex
	path      string
	checkedAt time.Time
	exists    bool
	modTime   time.Time
	name      string
}

func whaleAssistantNavLabel(m Model) string {
	fallback := i18n.T(i18n.KeySidebarAssistant)
	if !m.ConnectionReady {
		return fallback
	}
	path := filepath.Join(paths.DshHome(), "data", "whale", "settings.json")
	now := time.Now()
	whaleSettingsCache.mu.Lock()
	if whaleSettingsCache.path == path && now.Sub(whaleSettingsCache.checkedAt) < 2*time.Second {
		exists := whaleSettingsCache.exists
		name := whaleSettingsCache.name
		whaleSettingsCache.mu.Unlock()
		if exists && name != "" {
			return name
		}
		return fallback
	}
	whaleSettingsCache.mu.Unlock()

	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		whaleSettingsCache.mu.Lock()
		whaleSettingsCache.path = path
		whaleSettingsCache.checkedAt = now
		whaleSettingsCache.exists = false
		whaleSettingsCache.mu.Unlock()
		return fallback
	}
	whaleSettingsCache.mu.Lock()
	defer whaleSettingsCache.mu.Unlock()
	whaleSettingsCache.checkedAt = now
	whaleSettingsCache.exists = true
	if whaleSettingsCache.path != path || !whaleSettingsCache.modTime.Equal(st.ModTime()) {
		var cfg struct {
			Name string `json:"name"`
		}
		name := ""
		if raw, err := os.ReadFile(path); err == nil {
			if json.Unmarshal(raw, &cfg) == nil {
				name = strings.TrimSpace(cfg.Name)
			}
		}
		whaleSettingsCache.path = path
		whaleSettingsCache.modTime = st.ModTime()
		whaleSettingsCache.name = name
	}
	if whaleSettingsCache.name != "" {
		return whaleSettingsCache.name
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

// Model is sidebar state for one frame.
type Model struct {
	PublicContributions      pluginui.PublicContributions
	WorkspaceNavigation      []pluginui.WorkspaceNavigation
	OnWorkspaceSessionAction func(sessionID, owner, action string)
	ownerNavigation          map[string]pluginui.WorkspaceNavigation
	navigationOnly           map[string]bool
	ownerRow                 bool
	DirectoryActions         []workspacepicker.DirectoryAction
	NativePages              []pluginui.NativeView
	OnOpenNativePage         func(handle string)
	// ViewportW/H are the window's client area in DIPs. Context.Size reports
	// device pixels on a scaled display, so the hover cards take their bounds
	// from here and fall back to Context.Size only when the frame did not say.
	ViewportW                   float32
	ViewportH                   float32
	ConnectionReady             bool
	ConnectionError             string
	DesktopUpdateInstalling     bool
	DesktopUpdatePhase          string
	DesktopUpdateDetail         string
	OnOpenDesktopUpdate         func()
	ConnectionConnecting        bool
	ConnectionEpoch             uint64
	OnReconnect                 func()
	PendingStatuses             map[string]string
	sessionDrag                 *sessionDragContext
	sessionHover                *sessionHoverState
	Batch                       session.SessionBatch
	OnBatchEdit                 func(string, []session.SessionBatchTarget)
	OnBatchRequest              func(string)
	OnBatchConfirm              func(uint64)
	OnBatchCancel               func(uint64)
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
	ActivePanel        string
	NewSessionShortcut string
	MenuShortcutLabels map[string]string
	OnNoDirectory      func()
	OnOpenMarket       func()
	OnOpenSchedule     func()
	OnOpenAssistant    func()
	OnOpenLauncher     func()
	// Explicit native access is attached only to the selected Session row.
	OnOpenSessionPlugins  func(sessionID string)
	OnCloseSessionPlugins func(sessionID string)
	SessionPluginsActive  bool
	OnContactAccount      func()
	OnOpenSettings        func()
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

// sidebarBrand keeps the caption drag surface outside the non-macOS session button.
func sidebarBrand(c *ui.Context, m Model, colors theme.Colors, fg, muted ui.Color, platform string) {
	skinLayout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark)
	identity := func() {
		if skinLayout.Skin == theme.SkinCodex {
			publicSidebarSlot(c, m, "sidebar.brand.mark", "", nil)
			publicSidebarSlot(c, m, "sidebar.brand.name", "", func() {
				ui.Text(c, product.Name).Font(theme.RenderPreferences().Interface.Family).FontSize(15).FontWeight(600).LetterSpacing(-.15).Grow(1).MinWidth(0).SingleLine().TextColor(fg)
			})
			return
		}
		publicSidebarSlot(c, m, "sidebar.brand.mark", "", func() {
			if bmp := chrome.BrandAvatar(); bmp != nil {
				ui.Image(c, bmp).Size(38, 38).Fit(ui.Contain)
			} else {
				ui.Box(c).Size(38, 38).Radius(19).Background(chrome.Color(colors.Accent).Alpha(0.25))
			}
		})
		// Claude's brand scope uses the user's sans stack rather than Inter.
		publicSidebarSlot(c, m, "sidebar.brand.name", "", func() {
			ui.Column(c).Gap(1).Grow(1).MinWidth(0).Font(theme.RenderPreferences().Interface.Family).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).MinWidth(0).Children(func() {
					ui.Row(c).Label("鲸屿go").Gap(0).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "鲸").FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(20)).
							FontWeight(700).TextColor(fg).SingleLine()
						ui.Text(c, "屿go").FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(20)).
							FontWeight(700).TextColor(chrome.Color(colors.Accent)).SingleLine()
					})
					ui.Text(c, "WHALE ISLE").FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(14)).
						FontWeight(700).LetterSpacing(0.6).TextColor(fg.Alpha(0.88)).SingleLine()
				})
				ui.Text(c, "BASED ON DEEPSEEK HARNESS").FontSize(theme.InterfaceSize(8.5)).
					FixedLineHeight(theme.InterfaceSize(11)).FontWeight(500).LetterSpacing(0.28).TextColor(muted)
			})
		})
	}
	if platform == "darwin" || m.OnCreate == nil {
		ui.Row(c.Key("sidebar-brand-identity")).Label(product.Name).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Center).DragWindow().Children(identity)
		return
	}
	brand := ui.ButtonBase(c.Key("sidebar-brand-new")).Label(product.Name).Description(i18n.T(i18n.KeySidebarNewSessionFallback)).
		Grow(1).Shrink(1).MinWidth(0).Height(38).Gap(8).Padding(0).AlignItems(ui.Center).Justify(ui.Start).Radius(8)
	brand.Children(identity)
	if brand.Clicked() {
		m.OnCreate()
	}
}

var newSessionIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3"><path d="M2.37091 11.2501C1.58745 9.89288 1.32067 8.29835 1.61969 6.76006C1.91872 5.22177 2.76342 3.8433 3.99826 2.87846C5.2331 1.91362 6.77494 1.42737 8.33988 1.50925C9.90482 1.59113 11.3875 2.23562 12.5149 3.32406C13.6425 4.41269 14.3387 5.87206 14.4754 7.4334C14.612 8.99474 14.18 10.5529 13.2587 11.8209C12.3375 13.0888 10.9891 13.9813 9.46194 14.3337C8.18691 14.628 6.85895 14.5294 5.64989 14.0605C5.1712 13.8748 4.76962 13.4932 4.26534 13.3967C3.67413 13.2835 2.95257 13.5598 2.03794 14.3337"/><path d="M8 5V11M5 8H11"/></svg>`))

// The title stays centered over the full button while its right edge fades
// before the configured shortcut, matching SidebarRoot's label mask.
func newSessionButton(c *ui.Context, m Model, colors theme.Colors, fg, border ui.Color) {
	label := i18n.T(i18n.KeySidebarNewSessionFallback)
	if theme.ResolvedSkin(m.ThemeID, m.SystemDark) == theme.SkinCodex {
		btn := ui.Button(c.Key("sidebar-new-session"), "").Label(label).FillWidth().Height(32).Radius(8).Margin(2, 0, 6, 0).Padding(0, 8).BorderWidth(0).Background(ui.Transparent).Justify(ui.Start).Gap(8).TextColor(fg)
		if btn.Hovered() {
			btn.Background(chrome.Color(colors.Hover))
		}
		btn.Children(func() {
			ui.Icon(c, newSessionIcon).Size(16, 16).TextColor(fg)
			ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(fg)
		})
		if btn.Clicked() {
			m.OnCreate()
		}
		return
	}
	btnFill := chrome.Color(colors.Field)
	btnHoverFill := chrome.Color(colors.Hover)
	if m.SystemDark {
		btnFill = ui.RGBA(255, 255, 255, 0.18)
		btnHoverFill = ui.RGBA(255, 255, 255, 0.23)
	}
	btnBorder := chrome.Color(colors.BorderL3)
	if colors.BorderL3 == "" {
		btnBorder = border
	}
	btn := ui.Button(c.Key("sidebar-new-session"), "").Label(label).FillWidth().Height(38).Radius(12).
		Margin(4, 2, 0, 2).MaxWidth(252).Background(btnFill).Border(0.5, btnBorder).
		TextColor(fg).FontSize(theme.InterfaceSize(14)).FontWeight(500).Padding(8, 12, 8, 12)
	if btn.Hovered() {
		btn.Background(btnHoverFill)
	}
	showShortcut := m.NewSessionShortcut != "" && (btn.Hovered() || btn.FocusVisible())
	font := ui.Font{Family: c.Theme().Font, Size: theme.InterfaceSize(14), Weight: 500}
	title := ui.ShapeTextLine(label, font, 0)
	titleW, titleH := title.Size()
	shortcutW := float32(0)
	if showShortcut {
		shortcutW, _ = ui.ShapeTextLine(m.NewSessionShortcut, ui.Font{Family: font.Family, Size: font.Size, Weight: 400}, 0).Size()
	}
	btn.Children(func() {
		mask := ui.Box(c.Key("sidebar-new-label")).Grow(1).MinWidth(0).Height(22).PassThrough()
		mask.Draw(func(p *ui.Painter, r ui.Rect) {
			x := r.X + (r.W+shortcutW-titleW-20)/2
			draw := func(ink ui.Color) {
				p.Icon(newSessionIcon, ui.Rect{X: x, Y: r.Y + (r.H-14)/2, W: 14, H: 14}, ink)
				p.TextLayout(title, x+20, r.Y+(r.H-titleH)/2, ink)
			}
			if !showShortcut {
				p.Clip(r, 0, func() { draw(fg) })
				return
			}
			fade := min(float32(16), r.W)
			p.Clip(ui.Rect{X: r.X, Y: r.Y, W: r.W - fade, H: r.H}, 0, func() { draw(fg) })
			for step := float32(0); step < fade; step++ {
				p.Clip(ui.Rect{X: r.X + r.W - fade + step, Y: r.Y, W: min(float32(1), fade-step), H: r.H}, 0,
					func() { draw(fg.Alpha((fade - step - .5) / fade)) })
			}
		})
		if showShortcut {
			ui.Text(c, m.NewSessionShortcut).Role(ui.RoleNone).Width(shortcutW).Shrink(1).MinWidth(0).SingleLine().
				FontSize(font.Size).FontWeight(400).FixedLineHeight(22).TextColor(fg).PassThrough().
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
	m = workspaceOwnerNavigationModel(m)
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	if !m.SystemDark {
		// The rail's light glass over the wallpaper lifts muted gray below a
		// readable ratio; secondary ink keeps times and hints legible.
		muted = chrome.Color(colors.Secondary)
	}
	border := chrome.Color(colors.Border)
	// SidebarRoot.root: flex column height 100%, pad 6 / inline 12.
	// Children (flex:none): logoRow (60), newSession, panelList;
	// regionArea flex:1 min-height:0; footArea flex:none (pinned bottom).
	skinLayout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark)
	root := ui.Column(c.Key("sidebar-root")).Fill().MinWidth(0).Padding(6, skinLayout.SidebarInline, 6, skinLayout.SidebarInline)
	m = beginSessionHover(root, m)
	if skinLayout.Active() {
		root.FontSize(skinLayout.SidebarFont)
	}
	if m.Collapsed {
		// Native collapsed rail: SidebarRoot.module.css .root.collapsed is
		// `padding: 18px 10px 6px`, and the 36-DIP seats with their 12-DIP gaps
		// then reproduce the round261 UIA table (150%) exactly — 打开侧边栏 18,
		// 新建会话 66, 插件/自动化任务/鲸鱼娘 114/162/210, pitch 48. Keep the
		// reference constant: the uniform +0.67 the live window measured is a
		// window-origin artifact (round261 also saw native rects quantized to
		// 1/1.5 DIP: 9.33/35.33/36.67), not rail padding, so rounding it in
		// here would sit the rail 1 DIP high at 100% scaling.
		root.Padding(18, 10, 6, 10)
	}
	if m.SearchFocus == nil {
		m.SearchFocus = ui.Local(root, "search-focus-request", func() bool { return false })
	}
	// The archive filter belongs to the Host-scoped store, not this retained UI.
	// Only the independent management section's expansion is session-local.
	m.archiveExpanded = ui.Local(root, "archive-expanded", func() bool { return false })
	if m.PublicContributions.WorkspaceContext != nil {
		query := ""
		if m.SearchQuery != nil {
			query = *m.SearchQuery
		}
		m.PublicContributions.WorkspaceContext(query, *m.archiveExpanded)
	}
	m.searchState = ui.Local(root, "session-search", func() sessionSearchState {
		return sessionSearchState{Expanded: m.SearchQuery != nil && strings.TrimSpace(*m.SearchQuery) != ""}
	})
	m.workspaceWatermarks = ui.Local(root, "workspace-watermarks", func() workspaceWatermarkState { return workspaceWatermarkState{} })
	m.sidebarDrag = ui.Local(root, "sidebar-drag-state", func() sidebarDragState { return sidebarDragState{} })
	if skinLayout.Active() {
		fill := chrome.Color(colors.Sidebar)
		if m.Collapsed && skinLayout.Skin == theme.SkinCodex {
			fill = chrome.Color(colors.BG)
		}
		root.Background(fill).Radius(0).DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X + r.W - skinLayout.SidebarBorder, Y: r.Y, W: skinLayout.SidebarBorder, H: r.H}, chrome.Color(skinLayout.SidebarBorderColor), 0)
		})
	} else {
		root.Draw(chrome.Glass(chrome.Color(colors.Sidebar), chrome.MaterialSidebar))
	}
	root.Children(func() {
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
			// Keep the full product name readable within the sidebar's brand
			// column, alongside the avatar and collapse control.
			ui.Row(c).Key("sidebar-brand-row").FillWidth().Height(skinLayout.BrandHeight).Margin(0, 0, skinLayout.BrandBottom, 0).Gap(8).AlignItems(ui.Center).
				DragWindow().Children(func() {
				sidebarBrand(c, m, colors, fg, muted, runtime.GOOS)
				if m.OnToggle != nil {
					// Native 收起侧边栏: a 28x28 r8 control at (240,22) inside the
					// 60-DIP brand row; the 16x16 panel glyph sits centred.
					collapse := ui.Button(c, "").Label(i18n.T(i18n.KeySidebarCollapse)).Tooltip(i18n.T(i18n.KeySidebarCollapse) + " (Ctrl+B)").
						Width(28).Height(28).Padding(0).Radius(8).BorderWidth(0).Background(ui.Transparent).Center()
					collapseInk := chrome.Color(colors.Secondary)
					if collapse.Hovered() {
						collapse.Background(chrome.Color(colors.Hover))
						collapseInk = fg
					}
					collapse.Children(func() {
						ui.Icon(c, sidebarRailIcons["panel"]).Size(16, 16).TextColor(collapseInk)
						publicSidebarSlot(c, m, "sidebar.toggle.badge", "", nil)
					})
					if collapse.Clicked() {
						m.OnToggle()
					}
				}
			})
			if m.OnCreate != nil {
				// SidebarRoot.module.css .newSession: h38, radius 12, margin 0 2 8,
				// border-l2. The 4-DIP top margin is the page's 10-DIP gap under
				// the 60-DIP brand row (native: button at y=70, 252x38 at x=14).
				newSessionButton(c, m, colors, fg, border)
				// 12 DIP under the new-session button puts the panel list at the
				// native y=120 (button 70..108).
				if skinLayout.Skin != theme.SkinCodex {
					ui.Box(c).Height(12)
				}
			}
			// SidebarRoot.panelList: 36-high rows with a 4-DIP gap, so the
			// pitch is 40, inset 2 inside the 256-DIP list (native buttons sit at
			// x=14, 252 wide, with 插件 at y=120 and 自动化任务 at y=160).
			ui.Column(c).Label(sidebarLocaleText("panels.label")).FillWidth().Padding(0, 2, 8, 2).Gap(4).Children(func() {
				nav := func(id, key string, drawIcon func(), label string, fn func()) {
					row := ui.Button(c.Key("sidebar-panel-"+key), "").FillWidth().Height(skinLayout.NavHeight).Radius(12).
						Selected(m.ActivePanel == id).Background(ui.Transparent).TextColor(fg).BorderWidth(0).
						Padding(7, 8, 7, 8).AlignItems(ui.Center)
					if m.ActivePanel == id || row.Hovered() {
						row.Background(chrome.Color(colors.Hover))
					}
					if skinLayout.Skin == theme.SkinCodex {
						row.Padding(6, 8).MinHeight(32)
					}
					row = row.Transition(ui.ElementTransition{Duration: chrome.PopoverDuration, Colors: true})
					row.Children(func() {
						ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
							if drawIcon != nil {
								drawIcon()
							}
							// Native rows carry a 22-DIP line box under the 7-DIP
							// padding, so the label ink starts at 7 + (22-14)/2 below
							// the row; a plain text box is 18.7 tall and its centred
							// ink sits ~1.7 DIP lower than the reference.
							lineHeight := theme.InterfaceSize(22)
							if skinLayout.Active() {
								lineHeight = 20
							}
							ui.Text(c, label).FontSize(skinLayout.SidebarFont).
								FixedLineHeight(lineHeight).TextColor(fg).Grow(1)
						})
					})
					if fn != nil && row.Clicked() {
						fn()
					}
				}
				for _, panel := range sidebarPanelRows(m) {
					panel := panel
					nav(panel.active, panel.rowKey(), func() { publicPanelGlyph(c, m, panel, fg) }, panel.label, panel.run)
				}
			})
			menuOpen := &workspaceMenuFB
			if m.WorkspaceMenuOpen != nil {
				menuOpen = m.WorkspaceMenuOpen
			}
			// SidebarRoot.regionArea (flex:1): workspace browser + real session tree.
			publicSidebarSlot(c, m, "sidebar.workspaces", "", func() {
				ui.Column(c).FillWidth().Grow(1).MinHeight(0).Gap(6).Padding(8, 0, 0, 0).Children(func() {
					// WorkspaceBrowser.sectionHeader and Rows.tasksSection use the
					// source tertiary token, independent of other rail readability ink.
					sectionInk := chrome.Color(colors.Muted)
					workspaceHeading(c, m, menuOpen, fg, sectionInk)
					batchToolbar(c, m, fg)
					ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Gap(2).ScrollbarsVisible(sidebarScrollbarLinger(c, root)).Children(func() {
						if sessionSearchResultsView(c, m, fg, muted) {
							// Project 功能已改版，侧栏区块暂时下线；恢复时取消注释。
							// projects.View(c, m.Projects)
							publicSidebarSlot(c, m, "sidebar.workspaces.sections", "", nil)
							return
						}
						filter := browseArchiveFilter(m)
						items := sidebarItemsForArchiveFilter(m.Snapshot, m.SearchQuery, filter)
						workspaceRowsView(c, m, items, fg, muted, border)
						workspaceListFeedback(c, m, items, fg, muted)
						// Project 功能已改版，侧栏区块暂时下线；恢复时取消注释。
						// projects.View(c, m.Projects)
						archivedItems := archivedSectionItems(m)
						if (filter == "default" && !m.HideArchivedList) || len(archivedItems) > 0 {
							archivedSection(c, m, archivedItems, sectionInk, muted)
						}
						publicSidebarSlot(c, m, "sidebar.workspaces.sections", "", nil)
					})
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

			// SidebarRoot.footArea: one account seat for both signed-in and
			// signed-out profiles. Signed out it falls back to the settings
			// entry, keeping the account menu and person avatar.
			publicSidebarSlot(c, m, "sidebar.footer.action", "", nil)
			publicSidebarSlot(c, m, "sidebar.settings", "", func() {
				publicSidebarSlot(c, m, "settings.launcher", "", func() { accountRow(c, m, colors, muted, m.Collapsed) })
			})
			updatePhase := sidebarUpdatePhase(m)
			if !m.Collapsed && updatePhase != "" {
				bottomOffset := float32(4)
				if !accountSignedOut(m.Account) {
					bottomOffset = 8
				}
				ui.Box(c).Key("sidebar-footer-update-seat").Size(36, 36).Absolute().Right(2).Bottom(bottomOffset).Children(func() {
					desktopUpdateSeat(c, m, updatePhase, colors, false)
				})
			}
		})
	})
	renderSessionHover(c, m, root, m.sessionHover.dragging)
	sessionRenameDialog(c, m)
	batchDialog(c, m)
	batchResultToast(c, m)
}

func rowSession(c *ui.Context, it session.Item, depth int, m Model, fg, muted, border ui.Color) {
	m = workspaceOwnerRowModel(m, it.SessionID)
	if !m.ownerRow && batchSessionRow(c, m, it, depth, it.Title, fg, muted) {
		return
	}
	selected := it.SessionID == displayedSessionID(m)
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
	row := ui.Button(c, "").Label(label).Description(hint).FillWidth().Height(32).Radius(8).
		Background(ui.Transparent).BorderWidth(0).Padding(0, 8, 0, float32(8+depth*12)).Clip().AlignItems(ui.Center).Disabled(m.ownerRow && m.Batch.Enabled)
	revealSessionSearchRow(row, m, it.SessionID)
	if selected || row.Hovered() || row.FocusWithin() || row.MenuOpen() {
		row.Background(selectedBackground(colors))
	}
	menu := sessionRowMenu(m, it)
	pluginMenu := selected && m.OnOpenSessionPlugins != nil
	if (!it.Blank || pluginMenu) && !renaming {
		row.ContextMenu(menu)
	}
	hoverPressed := row.Pressed()
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(0).Children(func() {
			if status == "" && !it.Blank {
				if keepNative := publicSidebarBefore(c, m, "sidebar.session.row.leading", it.SessionID, "schedule-mark"); keepNative {
					if len(tasks) > 0 {
						sessionScheduleMark(c, len(tasks), muted)
					} else {
						sessionStatusSlot(c, status, muted)
					}
				}
				publicSidebarBefore(c, m, "sidebar.session.row.leading", it.SessionID, "")
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
				publicSidebarSlot(c, m, "sidebar.session.row.badge", it.SessionID, nil)
			}
			if !it.Blank || pluginMenu {
				showActions := row.Hovered() || row.FocusWithin() || row.MenuOpen()
				if !showActions && !it.Blank {
					ui.Text(c, sessionTrailingLabel(status, it.UpdatedAt)).Label("Session time " + it.SessionID).
						FontSize(theme.InterfaceSize(10)).FixedLineHeight(16).TextColor(muted).SingleLine()
					if it.Pinned {
						sessionPinIndicator(c, it.SessionID, muted)
					}
				}
				// Shipped rowActions: ellipsis, archive, pin, separated by 15 DIP.
				// Retain the menu seat for keyboard focus while the group is hidden.
				actions := ui.Row(c).Key("session-actions-" + it.SessionID).Shrink(0).AlignItems(ui.Center).Gap(15)
				if !showActions {
					actions.Absolute().Right(4).Opacity(0)
				}
				actions.Children(func() {
					more := ui.Button(c.Key("session-menu-"+it.SessionID), "").Label(sessionActionsLabel(it.SessionID, it.Title)).
						Tooltip(workspaceText("会话操作", "Session actions")).Width(16).Height(16).Padding(0).
						BorderWidth(0).Background(ui.Transparent).TextColor(muted).Center().Menu(menu)
					more.Children(func() { ui.Icon(c, menuEllipsisArtwork).Size(16, 16).TextColor(muted) })
					if showActions && !it.Blank {
						if publicSidebarBefore(c, m, "sidebar.workspaces.session.row.action", it.SessionID, "archive") {
							sessionRowAction(c, "archive-"+it.SessionID, menuArchiveArtwork, i18n.T(i18n.KeySidebarArchive),
								i18n.T(i18n.KeySidebarArchive), muted, archiveAction(m, it))
						}
						if publicSidebarBefore(c, m, "sidebar.workspaces.session.row.action", it.SessionID, "pin") {
							pinArt := menuPinArtwork
							if it.Pinned {
								pinArt = sessionPinArtwork
							}
							sessionRowAction(c, "pin-"+it.SessionID, pinArt, sessionPinLabel(it.Pinned),
								sessionPinTip(it.Pinned), muted, pinAction(m, it))
						}
						publicSidebarBefore(c, m, "sidebar.workspaces.session.row.action", it.SessionID, "")
					}
				})
			}
		})
	})
	if m.sessionHover != nil {
		m.sessionHover.dragging = m.sessionHover.dragging || row.Dragging()
	}
	publicSidebarRowContext(m, it.SessionID, row.MenuOpen(), row.Hovered() && !row.MenuOpen() && !m.Batch.Enabled && !renaming && !hoverPressed && (m.sessionHover == nil || !m.sessionHover.dragging))
	sessionHoverCard(c, m, row, it, label, status, colors, hoverPressed)
	if !renaming {
		sessionDragSource(row, m, it)
		sessionRowDropTarget(row, m, it, fg)
		if row.Clicked() && m.OnSelect != nil {
			m.OnSelect(it.SessionID)
		}
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
