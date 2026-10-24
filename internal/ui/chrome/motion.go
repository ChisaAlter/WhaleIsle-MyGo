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
	PopoverDuration = 160 * time.Millisecond
)

// How far a surface rises as it appears, in DIPs: a menu grows up out of its
// anchor by 4, a modal by 8, matching --ds-motion-distance-popover and
// --ds-motion-distance-overlay (ui-theme/src/styles/base.css:21-22).
const (
	popoverRise = 4
	overlayRise = 8
)

// PopoverTransition is the enter and exit motion of a menu, popover or floating
// card: it fades in from a few DIPs below its place and leaves the same way.
// Apply it to the panel element, which needs an ID stable across frames (a Key)
// for the framework to animate it:
//
//	ui.Column(c).Key("dock").Children(func() {
//		if open {
//			ui.Box(c).Key("dock-menu").Transition(chrome.PopoverTransition())
//		}
//	})
//
// A panel built by ui.PopoverBase cannot play it, since that base returns nil
// while closed and the framework enters an element only into a parent that was
// already there: use Reveal with PopoverDuration for a PopoverBase menu.
func PopoverTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: PopoverDuration,
		Ease:     ui.EaseInOut,
		Enter:    &ui.Motion{Y: popoverRise, Opacity: 0},
		Exit:     &ui.Motion{Y: popoverRise, Opacity: 0},
	}
}

// OverlayTransition is the enter and exit motion of a modal panel: a short fade
// with almost no travel, as motion.css's overlay recipe. The dialog frame
// (ui.DialogBase) supplies its own mounted lifetime, so apply this to the panel
// it hands over, with a Key.
//
// A DialogBase or PopoverBase panel cannot play it: those bases return nil while
// closed, and the framework skips Transition.Enter for an element whose parent
// is new in the same frame, which is exactly an overlay's first frame. Call
// Reveal for such a panel instead; it runs the same token by hand.
func OverlayTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: OverlayDuration,
		Ease:     ui.EaseInOut,
		Enter:    &ui.Motion{Y: overlayRise, Opacity: 0},
		Exit:     &ui.Motion{Y: overlayRise, Opacity: 0},
	}
}

// revealKey marks when an overlay opened, on the element carrying it.
type revealKey struct{}

// Reveal returns how far an overlay has faded in, from 0 to 1, and asks for the
// frames still to come. An overlay panel is painted with
//
//	progress := chrome.Reveal(c, backdrop, chrome.OverlayDuration)
//	chrome.RevealMask(backdrop, mask, progress)
//	panel.Opacity(progress)
//
// host is the element whose lifetime is one opening of the overlay: the backdrop
// or the panel, both of which DialogBase and PopoverBase rebuild for each
// opening and keep across the frames of that opening. Under reduced motion it
// returns 1 at once, so nothing moves.
//
// It exists because DialogBase and PopoverBase return nil while closed: the
// overlay, and so every element in it, is born in the frame it opens, and
// ElementTransition only enters an element into a parent that was already
// there. Fading the panel here gives the sheet's motion without a transition
// that never plays.
func Reveal(c *ui.Context, host ui.Element, d time.Duration) float32 {
	if d <= 0 {
		d = OverlayDuration
	}
	if c.Preferences().ReduceMotion {
		return 1
	}
	now := c.Now()
	if now.IsZero() {
		return 1
	}
	start := ui.Local(host, revealKey{}, func() time.Time { return now })
	if start.IsZero() || start.After(now) {
		*start = now
	}
	elapsed := now.Sub(*start)
	if elapsed >= d {
		return 1
	}
	c.AnimationFrame()
	return ui.EaseInOut(max(0, float32(elapsed)/float32(d)))
}

// RevealMask paints an overlay's dim mask over the whole window at progress, so
// it fades in with the panel instead of snapping on. Pass the dim color with its
// own alpha (the dialog masks use 0.24 in light and 0.5 in dark); progress
// scales it. Call it after Reveal in the same frame, on the backdrop Reveal
// measured.
func RevealMask(backdrop ui.Element, mask ui.Color, progress float32) {
	backdrop.Draw(func(p *ui.Painter, r ui.Rect) {
		if progress <= 0 || mask.A == 0 {
			return
		}
		p.Fill(r, mask.Alpha(progress), 0)
	})
}

// DialogMask is the dim layer under a modal card, `--dsw-alias-bg-mask-1`: .24
// over the light sheet and .5 over the dark one (`Modal.module.css:25-31`), so
// the card keeps its contrast on either scheme.
func DialogMask(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.RGBA(0, 0, 0, .5)
	}
	return ui.RGBA(0, 0, 0, .24)
}

// DialogReveal fades a hand-run DialogBase overlay in over the reference modal's
// 200ms enter (`Modal.module.css:29-31, 48`): the mask at the theme's depth and
// the panel at the same progress. Call it where the panel is styled, before its
// children. A tester that sets no clock fades with wall time, so a snapshot
// fixture settles the frame (SetTime twice) before it encodes the image.
func DialogReveal(c *ui.Context, backdrop, panel ui.Element) {
	progress := Reveal(c, backdrop, OverlayDuration)
	RevealMask(backdrop, DialogMask(c), progress)
	panel.Opacity(progress)
}
