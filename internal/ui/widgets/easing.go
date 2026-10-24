package widgets

import "github.com/egoist/mygo/ui"

// The design system's shared easing, cubic-bezier(0.4, 0, 0.2, 1)
// (ui-theme/src/styles/base.css:12 --ds-ease-in-out). The control points are
// asymmetric on x, so it is not the framework's cubic EaseInOut: it leaves
// slowly, is already far along by the middle (0.78 at half the time) and lands
// softly.
const (
	easeX1, easeY1 = 0.4, 0
	easeX2, easeY2 = 0.2, 1
)

// easeIterations solves the curve's x for a time fraction to about 1e-7 of a
// unit; halves of the remaining interval converge fast enough that more would
// only cost the frame.
const easeIterations = 24

// DesignEase is an easing of the toolkit's kind.
var _ ui.Easing = DesignEase

// DesignEase is the app's cubic-bezier(0.4, 0, 0.2, 1) easing, as ui.Easing:
// it maps the share of a duration already run, 0 to 1, to how far the motion
// has gone. CSS defines such a curve by its two control points and progresses
// the parameter with x, so the parameter is solved from the time fraction and
// the curve's y is read there.
//
// Values outside 0..1 clamp to the ends, and the result never decreases, so an
// animation along it only ever moves forward.
func DesignEase(t float32) float32 {
	switch {
	case !(t > 0): // also NaN
		return 0
	case t >= 1:
		return 1
	}
	lo, hi := float32(0), float32(1)
	s := t
	for i := 0; i < easeIterations; i++ {
		s = (lo + hi) / 2
		if easeAxis(s, easeX1, easeX2) < t {
			lo = s
		} else {
			hi = s
		}
	}
	return max(float32(0), min(float32(1), easeAxis(s, easeY1, easeY2)))
}

// easeAxis is one axis of a cubic Bezier from 0 to 1 through the two control
// values: 3(1-s)²s·p1 + 3(1-s)s²·p2 + s³.
func easeAxis(s, p1, p2 float32) float32 {
	u := 1 - s
	return 3*u*u*s*p1 + 3*u*s*s*p2 + s*s*s
}
