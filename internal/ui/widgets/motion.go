package widgets

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Motion pairs a duration with an easing curve, as the design system's motion
// tokens do: a surface fades rather than appears, a popover grows in place.
//
// Every primitive here animates through ui.Element.AnimateWith, which jumps
// straight to its target when the desktop asks for less motion
// (ui.Preferences.ReduceMotion), so a Motion never has to check that itself.
type Motion struct {
	// Duration is how long the motion takes.
	Duration time.Duration
	// Ease maps the time run, from 0 to 1, to how far the motion has gone.
	// Nil means EaseInOut.
	Ease ui.Easing
}

// The motion tokens. Overlay is a modal or a page, Popover a menu or a
// floating card, Swap a control changing its face (a chevron turning, a
// switch's thumb sliding) and Flip a whole surface turning over.
var (
	OverlayMotion = Motion{Duration: 200 * time.Millisecond, Ease: ui.EaseInOut}
	PopoverMotion = Motion{Duration: 150 * time.Millisecond, Ease: ui.EaseInOut}
	SwapMotion    = Motion{Duration: 120 * time.Millisecond, Ease: ui.EaseInOut}
	FlipMotion    = Motion{Duration: 300 * time.Millisecond, Ease: ui.EaseInOut}
)

// The durations of the motion tokens, for code that builds its own
// transitions rather than calling Motion.Animate.
const (
	// OverlayDuration is how long a modal or a page takes to appear.
	OverlayDuration = 200 * time.Millisecond
	// PopoverDuration is how long a menu or a floating card takes.
	PopoverDuration = 150 * time.Millisecond
	// SwapDuration is how long a control takes to change its face.
	SwapDuration = 120 * time.Millisecond
	// FlipDuration is how long a surface takes to turn over.
	FlipDuration = 300 * time.Millisecond
)

// Animate moves the value kept under key toward target as the motion says,
// starting from where it was, and requests frames while it moves. Values
// from 0 to 1 suit a fraction, a fade or a rotation in degrees over 360:
//
//	open := parts.Progress()
//	angle := widgets.SwapMotion.Animate(trigger, "chevron", open) * 180
func (m Motion) Animate(e ui.Element, key any, target float32) float32 {
	ease := m.Ease
	if ease == nil {
		ease = ui.EaseInOut
	}
	return e.AnimateWith(key, target, m.Duration, ease)
}
