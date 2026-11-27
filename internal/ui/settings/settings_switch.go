package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// settingsSwitchStyle used to pick between two installed switch looks. Both
// are coss's Switch now; the input style keeps its larger 44×28 hit area
// around the same track.
type settingsSwitchStyle uint8

const (
	settingsSwitchLabelled settingsSwitchStyle = iota
	settingsSwitchInput
)

// settingsPreferenceSwitch is coss's Switch on SwitchBase, which keeps the
// native toggle, focus, keyboard and Changed contract: a 30×18 track (input
// off, primary on), the 16 DIP background thumb sliding over 150 ms, and the
// ring-2 ring-offset-1 focus ring. All settings switches share the same 30×18
// dimensions so trailing controls align on one vertical baseline across rows.
func settingsPreferenceSwitch(c *ui.Context, value *bool, style settingsSwitchStyle) ui.Element {
	_ = style
	t := tk(c)
	switcher := ui.SwitchBase(c, value).Size(toggleWidth, toggleHeight).Shrink(0).Radius(design.RadiusFull).FocusRing(false).Cursor(ui.CursorPointer)
	handle := ui.Local(switcher, "switch-handle", func() ui.Handle { return ui.Handle{} })
	switcher.Bind(handle)
	position := switcher.AnimateWith("settings-switch-thumb", b01(*value), design.MotionFast, design.EaseInOut)
	off, on, thumb, ring := t.Input, t.Primary, switchThumbColor(t), t.Ring
	shadow := design.TW.Black.Alpha(0.05)
	switcher.DisabledOpacity(design.OpacityDisabled)
	switcher.Draw(func(p *ui.Painter, bounds ui.Rect) {
		track := ui.Rect{X: bounds.X + (bounds.W-toggleWidth)/2, Y: bounds.Y + (bounds.H-toggleHeight)/2, W: toggleWidth, H: toggleHeight}
		if el := c.Resolve(*handle); el.Valid() && el.FocusVisible() && !el.IsDisabled() {
			o := design.RingOffset + design.RingWidth
			outer := ui.Rect{X: track.X - o, Y: track.Y - o, W: track.W + 2*o, H: track.H + 2*o}
			p.Stroke(outer, ring, outer.H/2, design.RingWidth)
		}
		p.Fill(track, off.Mix(on, position), track.H/2)
		knob := ui.Rect{X: track.X + 1 + position*(toggleKnob-4), Y: track.Y + 1, W: toggleKnob, H: toggleKnob}
		for _, l := range design.ShadowSm {
			p.Shadow(knob, toggleKnob/2, l.X, l.Y, l.Blur, l.Spread, shadow)
		}
		p.Fill(knob, thumb, toggleKnob/2)
	})
	return switcher
}
