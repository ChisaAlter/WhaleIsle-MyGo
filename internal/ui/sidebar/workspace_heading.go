package sidebar

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// WorkspaceBrowser's section header contains search and selection in the same
// 36 DIP line. Expanding search takes the label and trailing actions' room.
func workspaceHeading(c *ui.Context, m Model, menuOpen *bool, fg, muted ui.Color) {
	header := ui.Row(c).FillWidth().Height(36).Gap(4).Padding(0, 0, 0, 4).AlignItems(ui.Center)
	expanded := ui.Local(header, "search-expanded", func() bool {
		return m.SearchQuery != nil && strings.TrimSpace(*m.SearchQuery) != ""
	})
	focus := m.SearchFocus != nil && *m.SearchFocus
	pendingFocus := ui.Local(header, "search-focus", func() bool { return false })
	focus = focus || *pendingFocus
	if focus {
		*expanded = true
	}
	header.Children(func() {
		if *expanded && m.SearchQuery != nil {
			input := ui.TextInput(c.Key("workspace-search"), m.SearchQuery).Label(i18n.T(i18n.KeySidebarSearch)).
				Placeholder(i18n.T(i18n.KeySidebarSearch)).Grow(1).MinWidth(0).Height(30)
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
				*expanded = false
			}
		} else {
			ui.Text(c, i18n.T(i18n.KeySidebarWorkspaceSection)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
			if m.SearchQuery != nil {
				search := ui.Button(c, "").Label(i18n.T(i18n.KeySidebarSearch)).Tooltip(i18n.T(i18n.KeySidebarSearch)).
					Width(28).Height(28).Padding(0).BorderWidth(0).Background(ui.Transparent)
				search.Draw(func(p *ui.Painter, r ui.Rect) {
					path := &ui.Path{}
					path.Circle(r.X+12, r.Y+12, 4.5)
					path.MoveTo(r.X+15.5, r.Y+15.5)
					path.LineTo(r.X+20, r.Y+20)
					p.StrokePath(path, 1.3, muted)
				})
				if search.Clicked() {
					*expanded = true
					*pendingFocus = true
					*menuOpen = false
				}
			}
		}
		batchToggle(c, m, muted)
		if !*expanded {
			options := ui.Button(c, "…").Label(workspaceText("工作区选项", "Workspace options")).Width(28).Height(28).
				Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted)
			if options.Clicked() {
				*menuOpen = !*menuOpen
			}
			ui.PopoverBase(c, options, menuOpen, func(panel ui.Element) {
				colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
				var rename func()
				if m.OnRenameWorkspace != nil {
					rename = func() { m.OnRenameWorkspace("") }
				}
				// The shared menu material: its own fill, the elevation hairline
				// and shadow, faded in over the popover token.
				progress := chrome.Reveal(c, panel, chrome.PopoverDuration)
				panel.Width(220).Padding(6).Gap(2).Radius(12).Opacity(progress)
				chrome.MenuSurface(panel, chrome.Color(colors.Field), chrome.Color(colors.Border), 12)
				panel.Children(func() {
					workspaceViewOptions(c, m)
					for _, action := range []struct {
						label string
						fn    func()
					}{
						{i18n.T(i18n.KeySidebarBrowseWorkspace), m.OnBrowseWorkspace},
						{i18n.T(i18n.KeySidebarRevealWorkspace), m.OnRevealWorkspace},
						{i18n.T(i18n.KeySidebarRenameWorkspace), rename},
						{i18n.T(i18n.KeySidebarDeleteWorkspace), m.OnDeleteWorkspace},
						{workspaceText("刷新会话", "Refresh sessions"), m.OnRefresh},
					} {
						if action.fn != nil {
							row := chrome.MenuRow(c, colors).Label(action.label)
							if row.Hovered() || row.Pressed() {
								row = row.Background(chrome.Color(colors.Hover))
							}
							if row.Clicked() {
								*menuOpen = false
								action.fn()
							}
						}
					}
				})
			})
			if m.OnBrowseWorkspace != nil && ui.Button(c, "+").Label(i18n.T(i18n.KeySidebarBrowseWorkspace)).Width(28).Height(28).
				Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
				m.OnBrowseWorkspace()
			}
		}
	})
}
