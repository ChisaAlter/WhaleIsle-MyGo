package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

// coss Switch geometry: a 30×18 track (thumb 16, 1 DIP inset), fully rounded.
const (
	toggleKnob   float32 = 4 * design.Unit
	toggleWidth  float32 = 2*toggleKnob - 2
	toggleHeight float32 = toggleKnob + 2
)

// settingsToggle paints one coss switch whose state the caller keeps: the
// element only reports the click, so the same row code works for host-backed
// and local flags. The track is input when off and primary when on, the
// thumb the background color with shadow-sm/5; keyboard focus shows coss's
// ring-2 ring-offset-1.
func settingsToggle(c *ui.Context, title string, value *bool, disabled bool) ui.Element {
	t := tk(c)
	on := value != nil && *value
	track := t.Input
	if on {
		track = t.Primary
	}
	toggle := ui.ButtonBase(c).Label(title).Width(toggleWidth).Height(toggleHeight).Shrink(0).
		Radius(design.RadiusFull).Padding(0).Gap(0).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer).Checked(on)
	if disabled {
		toggle = toggle.Disabled(true).DisabledOpacity(design.OpacityDisabled)
	}
	x := toggle.AnimateWith("settings-toggle-thumb", b01(on)*(toggleKnob-4), design.MotionFast, design.EaseInOut)
	focus := toggle.FocusVisible() && !disabled
	thumb, ring, gap := t.Background, t.Ring, t.Background
	shadow := design.TW.Black.Alpha(0.05)
	// The track and the thumb are painted, not laid out: a child box would take
	// the pointer hit before the button sees it and the switch would stop
	// toggling.
	toggle.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		if focus {
			p.Fill(ui.Rect{X: r.X - design.RingOffset - design.RingWidth, Y: r.Y - design.RingOffset - design.RingWidth, W: r.W + 2*(design.RingOffset+design.RingWidth), H: r.H + 2*(design.RingOffset+design.RingWidth)}, ring, r.H)
			p.Fill(ui.Rect{X: r.X - design.RingOffset, Y: r.Y - design.RingOffset, W: r.W + 2*design.RingOffset, H: r.H + 2*design.RingOffset}, gap, r.H)
		}
		p.Fill(r, track, r.H/2)
		knob := ui.Rect{X: r.X + 1 + x, Y: r.Y + 1, W: toggleKnob, H: toggleKnob}
		for _, l := range design.ShadowSm {
			p.Shadow(knob, toggleKnob/2, l.X, l.Y, l.Blur, l.Spread, shadow)
		}
		p.Fill(knob, thumb, toggleKnob/2)
	})
	return toggle
}

func b01(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

// settingsToggleRow is a preference row whose control is the switch.
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

// settingsSegmented is a page-level switch between two or more views: coss's
// SegmentedControl. It reports the picked index, -1 when none was picked this
// frame.
func settingsSegmented(c *ui.Context, key string, labels []string, active int, disabled bool) (ui.Element, int) {
	selected := active
	root := coss.SegmentedControl(c, &selected, labels, coss.SegmentedOptions{Disabled: disabled, Key: key})
	if root.Changed() && !disabled {
		return root, selected
	}
	return root, -1
}

// bodyToggleRowBadge is settingsToggleRow with a status badge after the title,
// used where a row is marked beta (机器人（Bots）· 测试中).
func bodyToggleRowBadge(c *ui.Context, key, title, badge, hint string, value *bool, disabled bool, fg, muted ui.Color, changed func(bool)) {
	if value == nil {
		return
	}
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	settingsRowRule(c, row)
	row.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(6)).Padding(design.Sp(4), 0).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1)).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
					kLabel(c, title, tk(c).Foreground)
					if badge != "" {
						coss.Badge(c, badge, coss.BadgeOptions{Variant: coss.BadgeOutline})
					}
				})
				if hint != "" {
					kHint(c, hint)
				}
			})
			ui.Row(c).Shrink(0).Justify(ui.End).AlignItems(ui.Center).Children(func() {
				if settingsToggle(c, key, value, disabled).Clicked() && !disabled {
					*value = !*value
					if changed != nil {
						changed(*value)
					}
				}
			})
		})
	})
}
