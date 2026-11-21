package settings

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// The installed Switch has a labelled button and an unlabelled native-input
// branch. The latter reserves a 44x28 hit area around the same 36x20 track.
// SwitchBase retains the native toggle, focus, keyboard and Changed contract.
type settingsSwitchStyle uint8

const (
	settingsSwitchLabelled settingsSwitchStyle = iota
	settingsSwitchInput
)

func settingsPreferenceSwitch(c *ui.Context, value *bool, style settingsSwitchStyle) ui.Element {
	switcher := ui.SwitchBase(c, value).Size(36, 20).Shrink(0).Radius(999).FocusRing(false)
	if style == settingsSwitchInput {
		switcher.Size(44, 28)
	}
	colors := theme.ResolveWithSystem(theme.IDSystem, c.Theme().Dark).Active(c.Theme().Dark)
	active, thumb := theme.Color(colors.BrandPrimary), theme.Color(colors.ButtonFG)
	if style == settingsSwitchInput {
		active, thumb = theme.Color(colors.Accent), theme.Color(colors.Field)
		if !c.Theme().Dark && colors.Field == "#f5f6f7" {
			thumb = ui.RGB(255, 255, 255) // bg-layer-1, not module-platform
		}
	}
	target := float32(0)
	if *value {
		target = 1
	}
	// The labelled switch's knob is the muted ink while it is off in the dark half
	// (0.3.13: rgb(173, 178, 184)) and slides to the button foreground when on.
	offThumb := thumb
	if style == settingsSwitchLabelled && c.Theme().Dark {
		offThumb = theme.Color(colors.Muted)
	}
	position := switcher.Animate("settings-switch-thumb", target, 100*time.Millisecond)
	switcher.Draw(func(p *ui.Painter, bounds ui.Rect) {
		track := ui.Rect{X: bounds.X + (bounds.W-36)/2, Y: bounds.Y + (bounds.H-20)/2, W: 36, H: 20}
		p.Fill(track, theme.Color(colors.BorderL3).Mix(active, position), 10)
		knob := ui.Rect{X: track.X + 2 + position*16, Y: track.Y + 2, W: 16, H: 16}
		p.Fill(knob, offThumb.Mix(thumb, position), 8)
		if switcher.FocusVisible() && !switcher.IsDisabled() {
			p.Stroke(ui.Rect{X: track.X - 4, Y: track.Y - 4, W: track.W + 8, H: track.H + 8}, theme.Color(colors.Accent), 14, 2)
		}
	})
	return switcher
}
