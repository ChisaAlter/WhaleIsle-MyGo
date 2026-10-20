package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// Toggle geometry from the installed build: a 44×24 track with a 20 DIP knob and
// 2 DIP inset, fully rounded. Settings paints its own switch so every row in this
// domain matches that size instead of the framework default.
const (
	toggleWidth  float32 = 44
	toggleHeight float32 = 24
	toggleKnob   float32 = 20
)

// settingsToggle paints one switch. The caller keeps the state: the element only
// reports the click, so the same row code works for host-backed and local flags.
func settingsToggle(c *ui.Context, title string, value *bool, disabled bool) ui.Element {
	on := value != nil && *value
	track := c.Theme().Border
	if on {
		track = c.Theme().Accent
	}
	toggle := ui.Button(c, "").Label(title).Width(toggleWidth).Height(toggleHeight).Shrink(0).
		Radius(toggleHeight / 2).BorderWidth(0).Padding(0).Gap(0).
		Background(ui.Transparent).AlignItems(ui.Center)
	if disabled {
		toggle = toggle.Disabled(true).Opacity(0.5)
	}
	// The track and the knob are painted, not laid out: a child box would take the
	// pointer hit before the button sees it and the switch would stop toggling.
	knob := ui.Hex("#ffffff")
	toggle.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		p.Fill(r, track, r.H/2)
		x := r.X + 2
		if on {
			x = r.X + r.W - 2 - toggleKnob
		}
		p.Fill(ui.Rect{X: x, Y: r.Y + (r.H-toggleKnob)/2, W: toggleKnob, H: toggleKnob}, knob, toggleKnob/2)
	})
	return toggle
}

// settingsToggleRow is a preference row whose control is the 44×24 switch.
func settingsToggleRow(c *ui.Context, key, title, hint string, value *bool, disabled bool, fg, muted ui.Color, changed func(bool)) {
	if value == nil {
		return
	}
	bodyPreferenceRow(c, key, title, hint, fg, muted, func() {
		if settingsToggle(c, title, value, disabled).Clicked() && !disabled {
			*value = !*value
			if changed != nil {
				changed(*value)
			}
		}
	})
}

// settingsSegmented is the two-or-more segment control the installed build uses
// for page-level switches: one track, equal segments, the active segment on a
// raised pill. It reports the picked index.
func settingsSegmented(c *ui.Context, key string, labels []string, active int, disabled bool) (ui.Element, int) {
	const (
		segmentHeight float32 = 38
		segmentWidth  float32 = 230
	)
	root := ui.Row(c.Key(key)).Height(segmentHeight).Shrink(0).Gap(2).Padding(3).
		Radius(segmentHeight / 2).Background(c.Theme().SurfaceHover).AlignItems(ui.Stretch)
	picked := -1
	root.Children(func() {
		for index, label := range labels {
			i, caption := index, label
			segment := ui.Button(c.Key(key+"-"+caption), "").Label(caption).
				Width(segmentWidth).Shrink(0).Padding(0).Radius((segmentHeight - 6) / 2).
				BorderWidth(0).Background(ui.Transparent).Disabled(disabled).
				AlignItems(ui.Center).Justify(ui.Center)
			if i == active {
				segment.Background(c.Theme().Surface)
			}
			segment.Children(func() {
				ui.Text(c, caption).FontSize(13).TextColor(c.Theme().Text)
			})
			if segment.Clicked() && !disabled {
				picked = i
			}
		}
	})
	return root, picked
}

// bodyToggleRowBadge is settingsToggleRow with a status badge after the title,
// used where the installed build marks a row as beta (机器人（Bots）· 测试中).
func bodyToggleRowBadge(c *ui.Context, key, title, badge, hint string, value *bool, disabled bool, fg, muted ui.Color, changed func(bool)) {
	if value == nil {
		return
	}
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	leadingRule := nextPreferenceRowIndex(c.Now()) > 0
	row.Draw(func(p *ui.Painter, bounds ui.Rect) {
		if leadingRule {
			p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y, W: bounds.W, H: .5}, ui.RGBA(muted.R, muted.G, muted.B, .25), 0)
		}
	})
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).MinHeight(64).Gap(24).Padding(16, 0).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
					if badge != "" {
						ui.Text(c, badge).Padding(1, 6).Radius(4).FontSize(theme.InterfaceSize(10)).
							Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .35)).TextColor(muted)
					}
				})
				if hint != "" {
					ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
				}
			})
			ui.Column(c).Width(180).Shrink(0).Gap(6).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.End).AlignItems(ui.Center).Children(func() {
					if settingsToggle(c, key, value, disabled).Clicked() && !disabled {
						*value = !*value
						if changed != nil {
							changed(*value)
						}
					}
				})
			})
		})
	})
}