package chrome

import (
	"math"
	"testing"
)

// TestWallpaperEffectSemantics pins the harness mapping from ui-theme's
// wallpaper layer: the blur slider is a percentage rendered as blur(p/100*40)px,
// the pixelate slider is a 1–20 scale factor, and both clamp to 0–100 with
// non-finite input reading as 0.
func TestWallpaperEffectSemantics(t *testing.T) {
	for _, tc := range []struct {
		percent float64
		blur    int
		factor  float64
	}{
		{0, 0, 1},
		{8, 3, 2.52},
		{12, 5, 3.28},
		{50, 20, 10.5},
		{100, 40, 20},
		{150, 40, 20},
		{-5, 0, 1},
		{math.NaN(), 0, 1},
		{math.Inf(1), 0, 1},
		{0.4, 0, 1},
		{99.6, 40, 20},
	} {
		if got := wallpaperBlurRadius(tc.percent); got != tc.blur {
			t.Fatalf("wallpaperBlurRadius(%v) = %d, want %d", tc.percent, got, tc.blur)
		}
		if got := wallpaperPixelFactor(tc.percent); math.Abs(got-tc.factor) > 1e-9 {
			t.Fatalf("wallpaperPixelFactor(%v) = %v, want %v", tc.percent, got, tc.factor)
		}
	}
	// The block size the shell paints is the rounded factor.
	for _, tc := range []struct {
		percent float64
		block   int
	}{
		{0, 1}, {8, 3}, {12, 3}, {50, 11}, {100, 20}, {150, 20},
	} {
		if got := int(math.Round(wallpaperPixelFactor(tc.percent))); got != tc.block {
			t.Fatalf("pixel block for %v = %d, want %d", tc.percent, got, tc.block)
		}
	}
}
