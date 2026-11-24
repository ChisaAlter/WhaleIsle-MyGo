package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// settingsRowWide is the content width from which a row puts its control
// beside its copy instead of under it.
const settingsRowWide = 105 * design.Unit

// settingsRowRule draws the row's 1 DIP border-colored bottom rule: rows of a
// page are a coss item list, each separated from the next.
func settingsRowRule(c *ui.Context, row ui.Element) {
	row.BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(tk(c).Border)
}

// settingsFieldRow paints one row of a schema-driven settings page: the title
// (coss Label) and its description on the left, the control on the right,
// separated from the next row by a rule; the optional reset action stays with
// the copy instead of competing with the control.
//
// The measured content width decides between the row layout and the stacked
// layout, so the same row keeps working inside the modal and in a narrow pane.
func settingsFieldRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, reset func(), controls func()) ui.Element {
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	help := takeRowHelp()
	settingsSearchTarget(c, key, title, row)
	settingsRowRule(c, row)
	width := row.Bounds().W
	wide := width >= settingsRowWide
	row.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if bounds.W != width {
			c.Invalidate()
		}
	})
	body := func() {
		text := ui.Column(c).MinWidth(0).Gap(design.Sp(1))
		if wide {
			text.Grow(1)
		} else {
			text.FillWidth()
		}
		text.Children(func() {
			preferenceRowTitle(c, title, fg, help)
			if hint != "" {
				kHint(c, hint)
			}
			if reset != nil {
				resetLabel := i18n.T(i18n.KeySurfacesSettingsReset)
				if coss.Button(c, resetLabel, coss.ButtonOptions{Variant: coss.ButtonLink, Size: coss.ButtonXS}).Tooltip(resetLabel).Clicked() {
					reset()
				}
			}
		})
		if controls == nil {
			return
		}
		// The control column ends where the row ends, so an intrinsic-width
		// control lines up with the list's right edge.
		control := ui.Column(c).MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.End)
		if wide {
			control.Width(design.Sp(52)).Shrink(0)
		} else {
			control.FillWidth()
		}
		control.Children(controls)
	}
	row.Children(func() {
		if wide {
			ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(6)).Padding(design.Sp(4), 0).AlignItems(ui.Center).Children(body)
		} else {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2.5)).Padding(design.Sp(4), 0).Children(body)
		}
	})
	return row
}

// settingsFieldBlock paints a control that needs the full column width, such as
// a multi-line text area or a JSON editor: the title and description stay above
// it, and the block keeps the same rule rhythm as settingsFieldRow.
func settingsFieldBlock(c *ui.Context, key, title, hint string, fg, muted ui.Color, control func()) ui.Element {
	block := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(4), 0)
	settingsSearchTarget(c, key, title, block)
	settingsRowRule(c, block)
	block.Children(func() {
		if title != "" {
			kLabel(c, title, tk(c).Foreground).FillWidth()
		}
		if hint != "" {
			kHint(c, hint)
		}
		if control != nil {
			control()
		}
	})
	return block
}

// settingsGroupTitle starts a group of rows inside one page.
func settingsGroupTitle(c *ui.Context, title string, fg ui.Color) {
	kHeading(c, title).Padding(design.Sp(6), 0, design.Sp(1), 0)
}

// settingsStatusLine is the small inline feedback line a settings page shows
// next to its save action.
func settingsStatusLine(c *ui.Context, text string, color ui.Color) {
	kText(c, text, design.TextXS, design.WeightNormal, color)
}
