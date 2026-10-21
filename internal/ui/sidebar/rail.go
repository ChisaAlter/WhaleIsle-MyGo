package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// SidebarRoot's settled rail: 36px controls centered in 56px, with 12px
// between controls/groups. The caller retains the root and bottom settings seat.
func collapsedRail(c *ui.Context, m Model, fg ui.Color, colors theme.Colors) {
	button := func(key, label string, icon *ui.SVG, enabled bool, action func()) ui.Element {
		b := ui.ButtonBase(c.Key(key)).Size(36, 36).Shrink(0).Padding(0).Radius(12).
			BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).
			Label(label).Tooltip(label).Disabled(!enabled)
		if b.Hovered() {
			b.Background(chrome.Color(colors.Hover))
		}
		if icon != nil {
			b.Children(func() { ui.Icon(c, icon).Size(18, 18).TextColor(fg) })
		}
		if b.Clicked() && action != nil {
			action()
		}
		return b
	}
	toggle := button("sidebar-rail-toggle", i18n.T(i18n.KeySidebarExpand), nil, m.OnToggle != nil, m.OnToggle).
		Tooltip(i18n.T(i18n.KeySidebarExpand) + " (Ctrl+B)")
	toggle.Children(func() {
		if !toggle.Hovered() && !toggle.Focused() {
			if mark := chrome.BrandAvatar(); mark != nil {
				ui.Image(c, mark).Size(38, 38).Fit(ui.Contain)
				return
			}
		}
		ui.Icon(c, sidebarRailIcons["panel"]).Size(18, 18).TextColor(fg)
	})
	ui.Box(c).Height(12).Shrink(0)
	if m.OnCreate != nil {
		button("sidebar-rail-new", i18n.T(i18n.KeySidebarNewSessionFallback), sidebarRailIcons["new"], true, m.OnCreate)
		ui.Box(c).Height(12).Shrink(0)
	}
	ui.Column(c.Key("sidebar-rail-panels")).FillWidth().Gap(12).Shrink(0).Children(func() {
		button("sidebar-rail-market", i18n.T(i18n.KeySidebarMarket), sidebarRailIcons["plugins"], m.OnOpenMarket != nil, m.OnOpenMarket)
		button("sidebar-rail-schedule", i18n.T(i18n.KeySidebarSchedule), sidebarRailIcons["clock"], m.OnOpenSchedule != nil, m.OnOpenSchedule)
		assistant := button("sidebar-rail-assistant", i18n.T(i18n.KeySidebarAssistant), nil, m.OnOpenAssistant != nil, m.OnOpenAssistant)
		assistant.Children(func() {
			if mark := chrome.BrandAvatar(); mark != nil {
				ui.Image(c, mark).Size(18, 18).Fit(ui.Contain)
			} else {
				ui.Text(c, "◉").TextColor(fg).FontSize(theme.InterfaceSize(14))
			}
		})
	})
	ui.Box(c).Height(12).Shrink(0)
	// WorkspaceBrowser keeps this region seat mounted; Add precedes Search in
	// its pinned DOM. Both reuse the existing owner callbacks, without a new menu.
	ui.Column(c.Key("sidebar-rail-workspaces")).FillWidth().Grow(1).MinHeight(0).Gap(12).Children(func() {
		if m.OnBrowseWorkspace != nil {
			button("sidebar-rail-add", i18n.T(i18n.KeySidebarBrowseWorkspace), sidebarRailIcons["add"], m.OnToggle != nil, func() {
				m.OnToggle()
				m.OnBrowseWorkspace()
			})
		}
		button("sidebar-rail-search", i18n.T(i18n.KeySidebarSearch), sidebarRailIcons["search"], m.OnToggle != nil && m.SearchQuery != nil, func() {
			// Only this explicit search gesture requests focus on expansion.
			// An unrelated layout/terminal change must not focus the input.
			*m.SearchFocus = true
			if m.WorkspaceMenuOpen != nil {
				*m.WorkspaceMenuOpen = false
			}
			m.OnToggle()
		})
	})
}

// Pinned e353e8e ui-primitives regular artwork. Brand/Assistant reuse the
// existing local brand occupant; no remote SVG or second icon executor.
var sidebarRailIcons = map[string]*ui.SVG{
	"panel":   ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M13.5 1.5H2.5C1.94772 1.5 1.5 1.94772 1.5 2.5V13.5C1.5 14.0523 1.94772 14.5 2.5 14.5H13.5C14.0523 14.5 14.5 14.0523 14.5 13.5V2.5C14.5 1.94772 14.0523 1.5 13.5 1.5Z"/><path d="M5.5 1.5V14.5"/></svg>`)),
	"new":     ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M2.37091 11.2501C1.58745 9.89288 1.32067 8.29835 1.61969 6.76006C1.91872 5.22177 2.76342 3.8433 3.99826 2.87846C5.2331 1.91362 6.77494 1.42737 8.33988 1.50925C9.90482 1.59113 11.3875 2.23562 12.5149 3.32406C13.6425 4.41269 14.3387 5.87206 14.4754 7.4334C14.612 8.99474 14.18 10.5529 13.2587 11.8209C12.3375 13.0888 10.9891 13.9813 9.46194 14.3337C8.18691 14.628 6.85895 14.5294 5.64989 14.0605C5.1712 13.8748 4.76962 13.4932 4.26534 13.3967C3.67413 13.2835 2.95257 13.5598 2.03794 14.3337"/><path d="M8 5V11M5 8H11"/></svg>`)),
	"plugins": ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M7.84457 5.06199C11.6605 4.93876 14.7962 6.14848 14.8484 7.76397C14.8875 8.97461 13.1838 10.0696 10.7215 10.5942"/><path d="M5.12742 8.07731C5.00419 4.26138 6.21391 1.12568 7.8294 1.07351C9.04004 1.03441 10.135 2.73808 10.6596 5.20037"/><path d="M8.02457 10.6802C4.20865 10.8034 1.07294 9.5937 1.02077 7.97821C0.981678 6.76758 2.68535 5.67262 5.14763 5.14798"/><path d="M10.7476 7.89535C10.8708 11.7113 9.66109 14.847 8.0456 14.8991C6.83496 14.9382 5.74 13.2346 5.21536 10.7723"/></svg>`)),
	"clock":   ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M8 14C11.3137 14 14 11.3137 14 8C14 4.68629 11.3137 2 8 2C4.68629 2 2 4.68629 2 8C2 11.3137 4.68629 14 8 14Z"/><path d="M8 4.31V8.46L11 10.08"/></svg>`)),
	"add":     ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M5.54492 2.06738C5.91034 2.06754 6.26318 2.20149 6.53711 2.44336L7.94043 3.68164V4.7998C7.71462 4.74105 7.50367 4.63139 7.32617 4.47461L5.87598 3.19238C5.78477 3.11185 5.66658 3.06754 5.54492 3.06738H2.94922C2.67322 3.06738 2.44946 3.29145 2.44922 3.56738V12.4326C2.44927 12.7087 2.67311 12.9326 2.94922 12.9326H12.9326C13.2086 12.9325 13.4326 12.7086 13.4326 12.4326V8.53613H14.4326V12.4326C14.4326 13.2609 13.7609 13.9325 12.9326 13.9326H2.94922C2.12083 13.9326 1.44927 13.261 1.44922 12.4326V3.56738C1.44946 2.73916 2.12094 2.06738 2.94922 2.06738H5.54492Z" fill="currentColor" stroke="none"/><path d="M9.75977 4.50208H14.5509M12.1492 6.89758L12.1492 2.10642"/></svg>`)),
	"search":  ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M6.58727 11.8586C9.55061 11.8586 11.9529 9.45637 11.9529 6.49304C11.9529 3.5297 9.55061 1.12744 6.58727 1.12744C3.62394 1.12744 1.22168 3.5297 1.22168 6.49304C1.22168 9.45637 3.62394 11.8586 6.58727 11.8586Z"/><path d="M10.2991 10.3933L14.7783 14.8725"/></svg>`)),
}
