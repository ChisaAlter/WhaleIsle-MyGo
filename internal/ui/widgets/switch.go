package widgets

import (
	"github.com/egoist/mygo/ui"
)

// The metrics of a Switch: a 36×20 track with 2 DIPs of padding around a 16
// DIP thumb, as the design system's switch has.
const (
	SwitchWidth   = 36
	SwitchHeight  = 20
	switchPadding = 2
	switchThumb   = 16
)

// Switch is a two-state toggle, 36×20 DIPs, that flips *on when clicked or
// when Space, Enter or a tap of assistive technology reaches it while it has
// the keyboard focus: the track takes the theme's accent color when on and a
// neutral one when off, and the thumb slides across. It takes a 2 DIP focus
// ring while the keyboard focus shows.
//
// Name it for assistive technology, as SwitchRow does with its title:
//
//	ui.Row(c).Children(func() {
//		ui.Text(c, "Wrap lines")
//		widgets.Switch(c, &app.wrap).Label("Wrap lines")
//	})
//
// A nil pointer shows an off switch that does nothing.
func Switch(c *ui.Context, on *bool) ui.Element {
	t := c.Theme()
	live := on != nil
	if !live {
		on = new(bool)
	}
	sw := ui.SwitchBase(c, on).Size(SwitchWidth, SwitchHeight).Radius(SwitchHeight / 2).Shrink(0).FocusRing(false)
	if !live {
		sw.Disabled(true)
	}
	// The thumb slides on the swap motion; without motion it is already there.
	pos := SwapMotion.Animate(sw, "thumb", boolF32(*on))
	track := t.Border.Mix(t.TextMuted, 0.7)
	face := ui.RGB(255, 255, 255)
	sw.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, track.Mix(t.Accent, pos), r.H/2)
		x := r.X + switchPadding + (r.W-2*switchPadding-switchThumb)*pos
		p.Fill(ui.Rect{X: x, Y: r.Y + (r.H-switchThumb)/2, W: switchThumb, H: switchThumb}, face, switchThumb/2)
	})
	sw.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if sw.FocusVisible() {
			ring := r.H + 2
			p.FocusRing(ui.Rect{X: r.X - 1, Y: r.Y - 1, W: r.W + 2, H: r.H + 2}, [4]float32{ring / 2, ring / 2, ring / 2, ring / 2})
		}
	})
	return sw
}

// SwitchRowOptions describes a SwitchRow.
type SwitchRowOptions struct {
	// Title is the setting's name, and names the switch for assistive
	// technology.
	Title string
	// Description is the line under the title that says what the setting
	// does.
	Description string
	// On is the setting, which the switch flips in place.
	On *bool
	// Disabled refuses input, as while a write is in flight.
	Disabled bool
	// Last drops the row's separator, at the end of a group.
	Last bool
	// OnChange is told the new value once the switch flips.
	OnChange func(on bool)
}

// SwitchRow is a SettingRow whose control is a Switch: the title and its
// description on the left, the switch on the right, as a boolean setting is
// laid out:
//
//	widgets.SwitchRow(c, widgets.SwitchRowOptions{
//		Title:       labels.CloseToTray,
//		Description: labels.CloseToTrayHint,
//		On:          &app.closeToTray,
//		OnChange:    func(on bool) { app.save("closeToTray", on) },
//	})
func SwitchRow(c *ui.Context, o SwitchRowOptions) ui.Element {
	return SettingRow(c, SettingRowOptions{
		Title:       o.Title,
		Description: o.Description,
		Last:        o.Last,
		Control: func() {
			sw := Switch(c, o.On).Label(o.Title).Disabled(o.Disabled)
			if o.On != nil && sw.Changed() && o.OnChange != nil {
				o.OnChange(*o.On)
			}
		},
	})
}
