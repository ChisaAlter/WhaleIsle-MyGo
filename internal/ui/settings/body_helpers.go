package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// bodyCard follows the settings feature cards, independently of the navigation shell.
func bodyCard(c *ui.Context, m Model, content func()) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(14, 16).Border(0.5, chrome.Color(colors.Border).Alpha(0.45)).Radius(12).Background(chrome.Color(colors.Field)).Children(content)
}

// bodyRosterRow keeps metadata above actions at narrow widths without a fixed label column.
func bodyRosterRow(c *ui.Context, title string, content func(), fg, muted ui.Color) {
	ui.Column(c.Key(title)).FillWidth().MinWidth(0).Gap(6).Padding(12, 0).Children(func() {
		ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
		content()
	})
}

// bodyDialog follows feature-owned settings editors and keeps the caller's draft
// alive until its explicit cancel/save callback acknowledges the operation.
func bodyDialog(c *ui.Context, open *bool, title string, content func(), options ...ui.DialogOptions) {
	bodyDialogFooter(c, open, title, content, nil, options...)
}

// bodyDialogFooter keeps the editor's actions outside its bounded scroll body.
// Short forms size to their content instead of reserving a 720 DIP blank panel.
func bodyDialogFooter(c *ui.Context, open *bool, title string, content, footer func(), options ...ui.DialogOptions) {
	bodyDialogFooterWidth(c, open, title, 380, content, footer, options...)
}

// bodyDialogFooterWidth lets a feature keep its original confirmation/editor
// width while sharing the same bounded body and resident action row.
func bodyDialogFooterWidth(c *ui.Context, open *bool, title string, width float32, content, footer func(), options ...ui.DialogOptions) {
	var dismissal ui.DialogOptions
	if len(options) > 0 {
		dismissal = options[0]
	}
	ui.DialogBaseWithOptions(c, open, dismissal, func(backdrop, panel ui.Element) {
		w, h := c.Size()
		maxHeight := max(0, h-48)
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), max(0, min(width, w-48)), 0).
			MaxHeight(maxHeight).Gap(20).Padding(0, 0, 24, 0).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Padding(22, 14, 12, 24).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(theme.InterfaceSize(16)).FontWeight(500).Grow(1).MinWidth(0)
				close := ui.Button(c, "").Label(providerCopy("关闭", "Close")).Disabled(dismissal.DismissDisabled).Size(28, 28).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent)
				close.Children(func() { ui.Icon(c, settingsIcons["CloseOutline"]).Size(16, 16) })
				if close.Clicked() {
					if dismissal.OnDismiss != nil {
						dismissal.OnDismiss()
					} else {
						*open = false
					}
				}
			})
			bodyHeight := maxHeight - 110
			if footer != nil {
				bodyHeight -= 52
			}
			ui.Scroll(c).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(max(0, bodyHeight)).Shrink(1).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(12).Padding(0, 24).Children(content)
			})
			if footer != nil {
				ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Padding(0, 24).Wrap().Gap(8).Justify(ui.End).AlignItems(ui.Center).Children(footer)
			}
		})
	})
}
