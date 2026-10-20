package widgets

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Contrast floors the primitives promise. Text keeps the web's 4.5:1 for
// normal sizes, and parts that are not text — a switch's track, a focus ring —
// the 3:1 of controls and their states.
const (
	// TextContrast is the least contrast text must have against what it is
	// on.
	TextContrast = 4.5
	// ControlContrast is the least contrast a control that is not text must
	// have against what it is on.
	ControlContrast = 3.0
)

// RelativeLuminance returns how bright a color is, from 0 to 1, as the web's
// contrast rules measure it. Surfaces that paint their own material, as chrome's
// glass does, check the contrast they leave for text with it.
func RelativeLuminance(c ui.Color) float64 {
	channel := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

// ContrastRatio returns how far apart two colors are, from 1 to 21, as the
// web's contrast rules measure them.
func ContrastRatio(a, b ui.Color) float64 {
	hi, lo := RelativeLuminance(a), RelativeLuminance(b)
	if hi < lo {
		hi, lo = lo, hi
	}
	return (hi + 0.05) / (lo + 0.05)
}

// readableOn returns white or near-black, whichever reads better on fill: the
// text of a filled button, which cannot rely on the theme's own text colors.
func readableOn(fill ui.Color) ui.Color {
	white := ui.RGB(255, 255, 255)
	dark := ui.RGB(15, 17, 21)
	if ContrastRatio(white, fill) >= ContrastRatio(dark, fill) {
		return white
	}
	return dark
}
