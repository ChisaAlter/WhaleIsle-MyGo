package chrome

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
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

func TestWallpaperBlurMaterialScaleAndNoEdgeHalo(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	if got := wallpaperBlurMaterial(wallpaperBlurRadius(0)).Radius; got != 0 {
		t.Fatalf("0%% blur radius = %v, want 0", got)
	}
	if got := wallpaperBlurMaterial(wallpaperBlurRadius(20)).Radius; got != 8 {
		t.Fatalf("20%% blur radius = %v, want 8", got)
	}
	if got := wallpaperBlurMaterial(wallpaperBlurRadius(100)).Radius; got != 40 {
		t.Fatalf("100%% blur radius = %v, want 40", got)
	}

	url := pngDataURL(t, color.RGBA{R: 30, G: 50, B: 80, A: 255}, 16, 16)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Transparent = true
	prefs.Blur = 100
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, true)
		ui.Box(c).Fill().Draw(paint)
	}, 96, 64)
	tt.Frame()
	img := tt.Image()
	center := img.RGBAAt(48, 32)
	if center.A != 255 || center.B <= center.R {
		t.Fatalf("unexpected blurred wallpaper center pixel: %+v", center)
	}
	for _, pt := range [][2]int{{0, 0}, {95, 0}, {0, 63}, {95, 63}} {
		px := img.RGBAAt(pt[0], pt[1])
		if absInt(int(px.R)-int(center.R)) > 1 || absInt(int(px.G)-int(center.G)) > 1 || absInt(int(px.B)-int(center.B)) > 1 || px.A != 255 {
			t.Fatalf("edge halo at %v: got %+v, center %+v", pt, px, center)
		}
	}
}
