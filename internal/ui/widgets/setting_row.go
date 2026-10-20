package widgets

import (
	"github.com/egoist/mygo/ui"
)

// The metrics of a setting row: a title at 14/22 over a description at 12/18,
// 16 DIPs of room above and below, a control column and a hairline between
// rows, as the design system's settings pages do.
const (
	settingRowTitleSize = 14
	settingRowTitleLine = 22
	settingRowDescSize  = 12
	settingRowDescLine  = 18
	settingRowPadding   = 16
	settingRowControl   = 180
	settingRowGap       = 24
	// settingRowStackWidth is the width under which the control moves below
	// the text, as a settings page gets narrow.
	settingRowStackWidth = 420
)

// SettingRowOptions describes a SettingRow.
type SettingRowOptions struct {
	// Title is the setting's name, 14 DIPs over a 22 DIP line.
	Title string
	// Description is the line under the title that says what the setting
	// does, 12 DIPs over 18.
	Description string
	// Control builds the control on the right, in a column of its own: a
	// Switch, a Select, a text field or a button. It moves below the text
	// when the row is narrower than 420 DIPs.
	Control func()
	// Last drops the row's separator, at the end of a group.
	Last bool
}

// SettingRow lays out one setting as a row: the title and its description on
// the left, the control on the right, 16 DIPs of room above and below and a
// hairline under it that Last drops at the end of a group. It is the one row
// layout settings pages use, so that every setting reads alike:
//
//	widgets.SettingRow(c, widgets.SettingRowOptions{
//		Title:       labels.Theme,
//		Description: labels.ThemeHint,
//		Control: func() {
//			ui.Select(c, &app.theme, themes).Label(labels.Theme)
//		},
//		Last: true,
//	})
func SettingRow(c *ui.Context, o SettingRowOptions) ui.Element {
	t := c.Theme()
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(0)
	width := root.Bounds().W
	root.Draw(func(_ *ui.Painter, r ui.Rect) {
		// The row stacks until it knows how wide it is.
		if r.W != width {
			c.Invalidate()
		}
	})
	root.Children(func() {
		text := func() {
			column := ui.Column(c).MinWidth(0).Gap(4)
			if o.Control != nil && width >= settingRowStackWidth {
				column.Grow(1)
			} else {
				column.FillWidth()
			}
			column.Children(func() {
				ui.Text(c, o.Title).FillWidth().FontSize(settingRowTitleSize).
					LineHeight(float32(settingRowTitleLine) / float32(settingRowTitleSize)).TextColor(t.Text)
				if o.Description != "" {
					ui.Text(c, o.Description).FillWidth().FontSize(settingRowDescSize).
						LineHeight(float32(settingRowDescLine) / float32(settingRowDescSize)).TextColor(t.TextMuted)
				}
			})
		}
		control := func() {
			if o.Control == nil {
				return
			}
			column := ui.Box(c).MinWidth(0).Gap(6)
			if width >= settingRowStackWidth {
				column.Width(settingRowControl).Shrink(0)
			} else {
				column.FillWidth()
			}
			column.Children(o.Control)
		}
		build := func() {
			text()
			control()
		}
		if o.Control != nil && width >= settingRowStackWidth {
			ui.Row(c).FillWidth().MinWidth(0).Gap(settingRowGap).Padding(settingRowPadding, 0).
				AlignItems(ui.Center).Children(build)
		} else {
			ui.Column(c).FillWidth().MinWidth(0).Gap(10).Padding(14, 0).Children(build)
		}
		if !o.Last {
			ui.Box(c).FillWidth().Height(0.5).Background(t.Border)
		}
	})
	return root
}
