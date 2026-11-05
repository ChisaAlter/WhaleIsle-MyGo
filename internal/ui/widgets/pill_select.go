package widgets

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// PillSelect is the harness select trigger (ui-primitives Select.module.css): a
// 36 DIP button on the selector fill that is as wide as its label, with a muted
// 14 DIP chevron, over a popup of 36 DIP items. radius is the 18 DIP pill of
// most settings rows, or the 12 DIP corner of the ui-chat enums.
//
// It keeps native select admission, focus, arrow-key navigation and Changed
// semantics; the returned element is the trigger.
func PillSelect(c *ui.Context, colors theme.Colors, value *string, options []string, radius float32) ui.Element {
	sel := ui.SelectBase(c, value)
	trigger := sel.Trigger.Height(36).Padding(7, 14).Gap(12).Radius(radius).BorderWidth(0).Justify(ui.SpaceBetween).
		Background(theme.Color(colors.SelectorFill)).TextColor(c.Theme().Text).AlignItems(ui.Center)
	if trigger.Hovered() {
		trigger.Background(c.Theme().SurfaceHover)
	}
	trigger.Children(func() {
		ui.Text(c, *value).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).SingleLine()
		ui.Box(c).Size(14, 14).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			var path ui.Path
			path.MoveTo(r.X+r.W*.25, r.Y+r.H*.38).LineTo(r.X+r.W*.5, r.Y+r.H*.6).LineTo(r.X+r.W*.75, r.Y+r.H*.38)
			p.StrokePath(&path, 1.2, theme.Color(colors.Muted))
		})
	})
	sel.Popup(func(panel ui.Element) {
		panel.Padding(4).Radius(12).Border(.5, c.Theme().Border).Background(c.Theme().Surface)
		for _, option := range options {
			item := sel.Item(option).FillWidth().Height(36).Padding(7, 12).Radius(8)
			if item.Highlighted() || option == *value {
				item.Background(c.Theme().SurfaceHover)
			}
			item.Children(func() {
				ui.Text(c, option).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).SingleLine()
			})
		}
	})
	return trigger
}
