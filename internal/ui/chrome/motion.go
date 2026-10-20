package chrome

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The motion tokens of the design system, matching internal/ui/widgets so a
// surface painted from chrome enters in step with the widget primitives
// (ui-theme/src/styles/motion.css: overlay, popover). The framework skips both
// under reduce motion, so a surface needs no switch of its own.
const (
	// OverlayDuration is how long a modal or a page takes to appear.
	OverlayDuration = 200 * time.Millisecond
	// PopoverDuration is how long a menu or a floating card takes.
	PopoverDuration = 150 * time.Millisecond
)

// How far a surface rises as it appears, in DIPs: a menu grows up out of its
// anchor, a modal barely moves.
const (
	popoverRise = 6
	overlayRise = 2
)

// PopoverTransition is the enter and exit motion of a menu, popover or floating
// card: it fades in from a few DIPs below its place and leaves the same way.
// Apply it to the panel element, which needs an ID stable across frames (a Key)
// for the framework to animate it:
//
//	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
//		panel.Key("workspace-menu").Transition(chrome.PopoverTransition())
//	})
func PopoverTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: PopoverDuration,
		Ease:     ui.EaseInOut,
		Enter:    &ui.Motion{Y: popoverRise, Opacity: 0},
		Exit:     &ui.Motion{Y: popoverRise / 2, Opacity: 0},
	}
}

// OverlayTransition is the enter and exit motion of a modal panel: a short fade
// with almost no travel, as motion.css's overlay recipe. The dialog frame
// (ui.DialogBase) supplies its own mounted lifetime, so apply this to the panel
// it hands over, with a Key.
func OverlayTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: OverlayDuration,
		Ease:     ui.EaseInOut,
		Enter:    &ui.Motion{Y: overlayRise, Opacity: 0},
		Exit:     &ui.Motion{Y: overlayRise, Opacity: 0},
	}
}
