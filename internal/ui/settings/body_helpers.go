package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// bodyCard follows the settings feature cards, independently of the navigation shell.
func bodyCard(c *ui.Context, m Model, content func()) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(12, 16).Border(1, chrome.Color(colors.Border)).Radius(12).Background(chrome.Color(colors.Field)).Children(content)
}

// bodyRosterRow keeps metadata above actions at narrow widths without a fixed label column.
func bodyRosterRow(c *ui.Context, title string, content func(), fg, muted ui.Color) {
	ui.Column(c.Key(title)).FillWidth().MinWidth(0).Gap(6).Padding(12, 0).Children(func() {
		ui.Text(c, title).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
		content()
	})
}

// bodyDialog follows feature-owned settings editors and keeps the caller's draft
// alive until its explicit cancel/save callback acknowledges the operation.
func bodyDialog(c *ui.Context, open *bool, title string, content func()) {
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		w, h := c.Size()
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), max(0, min(720, w-32)), 20).
			Height(max(0, min(720, h-32))).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, title).FontSize(theme.InterfaceSize(16)).Bold().Grow(1).MinWidth(0)
				if ui.Button(c, providerCopy("关闭", "Close")).Clicked() {
					*open = false
				}
			})
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() { ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(content) })
		})
	})
}
