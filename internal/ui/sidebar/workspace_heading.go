package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// workspaceHeadingLift raises the whole heading row (工作区/多选会话/视图选项/
// 添加工作区/搜索会话) to the native y: round182 measured MyGo 6.67 DIP low
// (native 250 vs 256.67 window-relative).
const workspaceHeadingLift float32 = 6.67

// WorkspaceBrowser's section header contains search and selection in the same
// 36 DIP line. Expanding search takes the label and trailing actions' room.
func workspaceHeading(c *ui.Context, m Model, menuOpen *bool, fg, muted ui.Color) {
	header := ui.Row(c).FillWidth().Height(36).Gap(4).Padding(0, 0, 0, 4).AlignItems(ui.Center).OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: -workspaceHeadingLift} })
	state := m.searchState
	if state == nil {
		state = ui.Local(header, "session-search", func() sessionSearchState {
			return sessionSearchState{Expanded: m.SearchQuery != nil && session.NormalizeSessionSearchQuery(*m.SearchQuery) != ""}
		})
	}
	expanded, pendingFocus := &state.Expanded, &state.Focus
	focus := m.SearchFocus != nil && *m.SearchFocus || *pendingFocus
	if focus {
		*expanded = true
	}
	header.Children(func() {
		if *expanded && m.SearchQuery != nil {
			searchRoot := ui.Row(c.Key("workspace-search-root")).Grow(1).MinWidth(0).Height(30).Gap(4).AlignItems(ui.Center)
			searchRoot.Children(func() {
				input := ui.TextInput(c.Key("workspace-search"), m.SearchQuery).Label(i18n.T(i18n.KeySidebarSearch)).
					Placeholder(i18n.T(i18n.KeySidebarSearch)).Grow(1).MinWidth(0).Height(30)
				if input.Changed() {
					*m.SearchQuery = session.SanitizeSessionSearchInput(*m.SearchQuery)
				}
				if focus {
					input.Focus()
					*pendingFocus = false
					if m.SearchFocus != nil {
						*m.SearchFocus = false
					}
				}
				clear := ui.Button(c, "×").Label(workspaceText("清除搜索", "Clear search")).Width(24).Height(28).
					Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
				if input.Shortcut(0, ui.KeyEscape) || clear.Clicked() {
					*m.SearchQuery = ""
					*expanded, *pendingFocus = false, false
				}
			})
			// The source outside-click boundary excludes the adjacent selection
			// toggle. Keep a nonempty filter and ignore the opening focus gesture.
			if !focus && session.NormalizeSessionSearchQuery(*m.SearchQuery) == "" && searchRoot.PressedOutside() {
				*expanded = false
			}
		} else {
			labelSize := theme.InterfaceSize(14)
			if layout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark); layout.Active() {
				labelSize = layout.SidebarFont
			}
			sectionLabel := i18n.T(i18n.KeySidebarWorkspaceSection)
			if m.Snapshot.WorkspaceLayout.GroupBy == "flat" {
				sectionLabel = workspaceText("会话", "Sessions")
			}
			ui.Text(c, sectionLabel).FontSize(labelSize).FontWeight(400).FixedLineHeight(theme.InterfaceSize(20)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
			if m.SearchQuery != nil {
				search := ui.Button(c, "").Label(i18n.T(i18n.KeySidebarSearch)).Tooltip(i18n.T(i18n.KeySidebarSearch)).
					Width(28).Height(28).Padding(0).Radius(8).BorderWidth(0).Background(ui.Transparent).Center()
				searchInk := muted
				if search.Hovered() {
					search.Background(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Hover))
					searchInk = fg
				}
				search.Children(func() { ui.Icon(c, sidebarRailIcons["search"]).Size(14, 14).TextColor(searchInk) })
				if search.Clicked() {
					*expanded = true
					*pendingFocus = true
					*menuOpen = false
				}
			}
		}
		batchToggle(c, m, muted)
		if !*expanded {
			colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
			label := workspaceText("视图选项", "View options")
			options := ui.Button(c, "").Label(label).Tooltip(label).Width(28).Height(28).
				Padding(0).Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Center()
			optionsInk := muted
			if *menuOpen || options.Hovered() {
				options.Background(chrome.Color(colors.Hover))
				optionsInk = fg
			}
			options.Children(func() { ui.Icon(c, workspaceViewOptionsArtwork).Size(16, 16).TextColor(optionsInk) })
			if options.Clicked() {
				*menuOpen = !*menuOpen
			}
			ui.PopoverBase(c, options, menuOpen, func(panel ui.Element) {
				// The shared menu material: its own fill, the elevation hairline
				// and shadow, faded in over the popover token.
				bg := chrome.Color(colors.BG)
				dark := bg.R < 128 && bg.G < 128 && bg.B < 128
				progress := chrome.Reveal(c, panel, chrome.PopoverDuration)
				panel.Width(200).Padding(4).Gap(0).Radius(16).Role(ui.RoleMenu).Opacity(progress)
				chrome.MenuSurface(panel, chrome.MenuBaseFill(colors, dark), chrome.MenuBorderColor(colors, dark), 16)
				panel.Children(func() {
					workspaceViewOptionsMenu(c, m, menuOpen)
				})
			})
			if m.OnBrowseWorkspace != nil {
				add := ui.Button(c, "").Label(i18n.T(i18n.KeySidebarBrowseWorkspace)).Size(28, 28).Padding(0).Radius(8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Center()
				addInk := muted
				if add.Hovered() {
					add.Background(chrome.Color(colors.Hover))
					addInk = fg
				}
				add.Children(func() { ui.Icon(c, sidebarRailIcons["add"]).Size(16, 16).TextColor(addInk) })
				if add.Clicked() {
					m.OnBrowseWorkspace()
				}
			}
		}
	})
}
