package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// settingsFieldRow paints one row of a schema-driven settings page: the title
// and its description sit on the left, the control on the right, the row is
// separated from the previous one by a hairline, and the optional reset action
// stays with the copy instead of competing with the control.
//
// The measured content width decides between the row layout and the stacked
// layout, so the same row keeps working inside the modal and in a narrow pane.
func settingsFieldRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, reset func(), controls func()) ui.Element {
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	width := row.Bounds().W
	leadingRule := nextPreferenceRowIndex(c.Now()) > 0
	row.Draw(func(p *ui.Painter, bounds ui.Rect) {
		if leadingRule {
			p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y, W: bounds.W, H: .5}, ui.RGBA(muted.R, muted.G, muted.B, .25), 0)
		}
		if bounds.W != width {
			c.Invalidate()
		}
	})
	body := func() {
		text := ui.Column(c).MinWidth(0).Gap(4)
		if width >= 420 {
			text.Grow(1)
		} else {
			text.FillWidth()
		}
		text.Children(func() {
			ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
			if hint != "" {
				ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
			}
			if reset != nil {
				resetLabel := i18n.T(i18n.KeySurfacesSettingsReset)
				if ui.Button(c, resetLabel).FontSize(theme.InterfaceSize(11)).BorderWidth(0).Padding(0).Background(ui.Transparent).TextColor(muted).Tooltip(resetLabel).Clicked() {
					reset()
				}
			}
		})
		if controls == nil {
			return
		}
		control := ui.Column(c).MinWidth(0).Gap(6)
		if width >= 420 {
			control.Width(180).Shrink(0)
		} else {
			control.FillWidth()
		}
		control.Children(controls)
	}
	row.Children(func() {
		if width >= 420 {
			ui.Row(c).FillWidth().MinWidth(0).Gap(24).Padding(16, 0).AlignItems(ui.Center).Children(body)
		} else {
			ui.Column(c).FillWidth().MinWidth(0).Gap(10).Padding(14, 0).Children(body)
		}
	})
	return row
}

// settingsFieldBlock paints a control that needs the full column width, such as
// a multi-line text area or a JSON editor: the title and description stay above
// it, and the block keeps the same hairline rhythm as settingsFieldRow.
func settingsFieldBlock(c *ui.Context, key, title, hint string, fg, muted ui.Color, control func()) ui.Element {
	block := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(8).Padding(14, 0)
	leadingRule := nextPreferenceRowIndex(c.Now()) > 0
	block.Draw(func(p *ui.Painter, bounds ui.Rect) {
		if leadingRule {
			p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y, W: bounds.W, H: .5}, ui.RGBA(muted.R, muted.G, muted.B, .25), 0)
		}
	})
	block.Children(func() {
		if title != "" {
			ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(fg)
		}
		if hint != "" {
			ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		}
		if control != nil {
			control()
		}
	})
	return block
}

// settingsGroupTitle starts a group of rows inside one page.
func settingsGroupTitle(c *ui.Context, title string, fg ui.Color) {
	ui.Text(c, title).FillWidth().Padding(16, 0, 6, 0).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(fg)
}

// settingsStatusLine is the small inline feedback line a settings page shows
// next to its save action.
func settingsStatusLine(c *ui.Context, text string, color ui.Color) {
	ui.Text(c, text).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(16)).TextColor(color)
}
