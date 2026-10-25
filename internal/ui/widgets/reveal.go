package widgets

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// revealKey marks when an overlay opened, on the element carrying it.
type revealKey struct{}

// Reveal returns how far an overlay has faded in, from 0 to 1, and asks for the
// frames still to come. An overlay panel is painted with
//
//	progress := widgets.Reveal(c, backdrop, widgets.OverlayDuration)
//	widgets.RevealMask(backdrop, mask, progress)
//	panel.Opacity(progress)
//
// host is the element whose lifetime is one opening of the overlay: the backdrop
// or the panel, both of which ui.DialogBase and ui.PopoverBase rebuild for each
// opening and keep across the frames of that opening. Under reduced motion it
// returns 1 at once, so nothing moves.
//
// It exists because DialogBase and PopoverBase return nil while closed: the
// overlay, and so every element in it, is born in the frame it opens, and
// ui.ElementTransition only enters an element into a parent that was already
// there. Fading the panel here gives the overlay's motion without a transition
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
	return DesignEase(max(0, float32(elapsed)/float32(d)))
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

// RevealOverlay fades an overlay's backdrop, dimmed at mask, and its panel in
// together over the overlay token, and returns the progress for anything else
// that enters with them. This is what a DialogBase panel plays when it keeps its
// own metrics: the mask it would have set with Background is painted at the same
// color and alpha, so only the timing changes.
func RevealOverlay(c *ui.Context, backdrop, panel ui.Element, mask ui.Color) float32 {
	progress := Reveal(c, backdrop, OverlayDuration)
	RevealMask(backdrop, mask, progress)
	panel.Opacity(progress)
	return progress
}
