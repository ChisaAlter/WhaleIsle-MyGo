package widgets

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
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
	// Nil means DesignEase.
	Ease ui.Easing
}

// The motion tokens. Overlay is a modal or a page, Popover a menu or a
// floating card, Swap a control changing its face (a chevron turning, a
// switch's thumb sliding) and Flip a whole surface turning over. All ride the
// design system's shared curve (ui-theme/src/styles/base.css:12).
//
// The durations are the reference token sheet's own values
// (ui-theme/src/styles/base.css:13-20):
//
//	--ds-transition-duration       0.2s  overlay
//	--ds-transition-duration-fast  0.1s  swap (--ds-motion-duration-swap)
//	--ds-motion-duration-popover   0.16s popover
//	--ds-motion-duration-flip      0.4s  flip
var (
	OverlayMotion = Motion{Duration: OverlayDuration, Ease: DesignEase}
	PopoverMotion = Motion{Duration: PopoverDuration, Ease: DesignEase}
	SwapMotion    = Motion{Duration: SwapDuration, Ease: DesignEase}
	SlowMotion    = Motion{Duration: SlowDuration, Ease: DesignEase}
	FlipMotion    = Motion{Duration: FlipDuration, Ease: DesignEase}
)

// FastDuration is the design system's fast token, --ds-transition-duration-fast
// (dsh-webui-tokens.css:12, 0.1s): what a control's own face takes to change
// under the pointer, as launcher.css:471 does for background
// (`transition: background var(--ds-transition-duration-fast)
// var(--ds-ease-in-out)`).
const FastDuration = design.MotionFast

// HoverMotion is a control's face changing under the pointer: the fast token
// along the design curve.
var HoverMotion = Motion{Duration: FastDuration, Ease: DesignEase}

// HoverTransition is the transition a control asks for so its hover and press
// fill follows the pointer instead of snapping: only the colors move, at the
// fast token, along the design curve. A button keeps its layout while its face
// changes, so neither position nor size animates.
func HoverTransition() ui.ElementTransition {
	return ui.ElementTransition{Duration: FastDuration, Ease: DesignEase, Colors: true}
}

// CardHoverTransition is the border/background transition for interactive cards
// over the popover token (160ms) along the design curve (docs/motion.md).
func CardHoverTransition() ui.ElementTransition {
	return ui.ElementTransition{Duration: PopoverDuration, Ease: DesignEase, Colors: true}
}

// The durations of the motion tokens, for code that builds its own
// transitions rather than calling Motion.Animate. All derive from design.Motion*
// so the 5-tier sheet has one shared source of truth across packages.
const (
	// DefaultDuration is the design system's base transition duration,
	// --ds-transition-duration (0.2s).
	DefaultDuration = design.MotionDefault
	// OverlayDuration is how long a modal or a page takes to appear.
	OverlayDuration = DefaultDuration
	// OverlayExitDuration is how long a modal takes to leave, deriving from
	// the fast token (--ds-motion-duration-overlay-out, 0.1s).
	OverlayExitDuration = FastDuration
	// PopoverDuration is how long a menu or a floating card takes.
	PopoverDuration = design.MotionPopover
	// SwapDuration is how long a control takes to change its face: the
	// reference's swap token is the fast token, not a duration of its own
	// (base.css:19 `--ds-motion-duration-swap: var(--ds-transition-duration-fast)`).
	SwapDuration = FastDuration
	// SlowDuration is the design system's slow track duration,
	// --ds-transition-duration-slow (0.3s), shared by sidebar, inspector, and
	// draft composer tracks.
	SlowDuration = design.MotionSlow
	// FlipDuration is how long a surface takes to turn over
	// (base.css:20 `--ds-motion-duration-flip: 0.4s`).
	FlipDuration = design.MotionFlip
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
		ease = DesignEase
	}
	return e.AnimateWith(key, target, m.Duration, ease)
}
