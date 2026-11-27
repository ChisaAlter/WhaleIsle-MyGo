package ui

import "math"

// FadeHorizontal clips painting to the element's horizontal bounds and fades
// its drawings and descendants from transparent at each edge to opaque left
// and right DIPs inward. Zero disables that edge; negative or nonfinite values
// disable it too. Fades wider than the element meet proportionally in its
// middle. Layout, focus and pointer bounds do not change.
//
// As with Opacity, the alpha multiplies each drawing; overlapping descendants
// do not first composite into an offscreen group. No background is painted.
func (e *node) FadeHorizontal(left, right float32) *node {
	e.fade = [2]float32{fadeWidth(left), fadeWidth(right)}
	return e
}

// FadeOverflowRight clips painting insetRight DIPs inside the element's right
// edge and linearly fades alpha to transparent over fadeRight DIPs inward from
// that clip edge. For text elements whose laid-out lines do not reach the inset
// right edge and are not truncated, the right fade is suppressed so short text
// stays crisp. Negative or nonfinite values are treated as zero. Layout, focus
// and pointer bounds do not change.
func (e *node) FadeOverflowRight(insetRight, fadeRight float32) *node {
	e.fadeInset[1] = fadeWidth(insetRight)
	e.fade[1] = fadeWidth(fadeRight)
	e.fadeOverflow = true
	return e
}

func fadeWidth(width float32) float32 {
	if width <= 0 || math.IsNaN(float64(width)) || math.IsInf(float64(width), 0) {
		return 0
	}
	return width
}
