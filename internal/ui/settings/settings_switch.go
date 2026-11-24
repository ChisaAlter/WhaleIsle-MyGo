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
// ring-2 ring-offset-1 focus ring.
func settingsPreferenceSwitch(c *ui.Context, value *bool, style settingsSwitchStyle) ui.Element {
	t := tk(c)
	switcher := ui.SwitchBase(c, value).Size(toggleWidth, toggleHeight).Shrink(0).Radius(design.RadiusFull).FocusRing(false).Cursor(ui.CursorPointer)
	if style == settingsSwitchInput {
		switcher.Size(11*design.Unit, 7*design.Unit)
	}
	position := switcher.AnimateWith("settings-switch-thumb", b01(*value), design.DurationDefault, design.EaseInOut)
	focus := switcher.FocusVisible() && !switcher.IsDisabled()
	off, on, thumb, ring, gap := t.Input, t.Primary, t.Background, t.Ring, t.Background
	shadow := design.TW.Black.Alpha(0.05)
	if switcher.IsDisabled() {
		switcher.DisabledOpacity(design.OpacityDisabled)
	}
	switcher.Draw(func(p *ui.Painter, bounds ui.Rect) {
		track := ui.Rect{X: bounds.X + (bounds.W-toggleWidth)/2, Y: bounds.Y + (bounds.H-toggleHeight)/2, W: toggleWidth, H: toggleHeight}
		if focus {
			o := design.RingOffset + design.RingWidth
			p.Fill(ui.Rect{X: track.X - o, Y: track.Y - o, W: track.W + 2*o, H: track.H + 2*o}, ring, track.H)
			g := design.RingOffset
			p.Fill(ui.Rect{X: track.X - g, Y: track.Y - g, W: track.W + 2*g, H: track.H + 2*g}, gap, track.H)
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
