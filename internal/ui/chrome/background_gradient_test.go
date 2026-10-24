package chrome

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

// TestDefaultGradientIsASoftFullBleedWash covers the Pass 2 backdrop: the
// installed build's sampled wash (wallpapers.PresetColors("dune")), full bleed
// at every corner, bright in the middle, and soft — the blurred shadow casts the
// old "aurora" blooms used left hard-edged circles over the wallpaper.
func TestDefaultGradientIsASoftFullBleedWash(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	// An explicit gradient choice (first run is flat now): the sampled wash is
	// what this test covers, not the default effect.
	prefs := theme.FirstRunAppearance()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter("", prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
	}, 320, 220)
	// A still frame: the wash drifts over time, samples would not repeat.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	image := tt.Image()

	// Full bleed: every corner carries the wash, not the bare theme background.
	background := theme.Resolve(theme.DefaultID).Light.BG
	page := Color(background)
	for _, pt := range [][2]int{{2, 2}, {317, 2}, {2, 217}, {317, 217}} {
		c := image.RGBAAt(pt[0], pt[1])
		if c.R == page.R && c.G == page.G && c.B == page.B {
			t.Fatalf("corner %v is the bare page colour %v, not the wash", pt, page)
		}
		if c.R > 250 && c.G > 250 && c.B > 250 {
			t.Fatalf("corner %v is white: the wash does not reach it", pt)
		}
	}

	// Soft: neighbouring pixels differ by a few levels at most, so there is no
	// circle edge anywhere in the window.
	worst, at := 0, [2]int{}
	for y := 0; y < image.Bounds().Dy(); y++ {
		for x := 1; x < image.Bounds().Dx(); x++ {
			a, b := image.RGBAAt(x-1, y), image.RGBAAt(x, y)
			for _, d := range []int{absInt(int(a.R) - int(b.R)), absInt(int(a.G) - int(b.G)), absInt(int(a.B) - int(b.B))} {
				if d > worst {
					worst, at = d, [2]int{x, y}
				}
			}
		}
	}
	if worst > 12 {
		t.Fatalf("the wash has a %d-level step at %v: hard edge", worst, at)
	}

	// The sampled palette: the middle is the bright centre of the reference, the
	// top is the cooler wash, and the lower left carries the green. The blooms
	// drift, so the green is looked for by hue rather than at a fixed pixel.
	top, middle := image.RGBAAt(160, 4), image.RGBAAt(160, 110)
	if middle.R <= top.R || middle.G <= top.G {
		t.Fatalf("centre %v is not brighter than the top %v", middle, top)
	}
	best, green := -1000.0, image.RGBAAt(0, 0)
	greenest := func(c color.RGBA) float64 { return float64(c.G) - (float64(c.R)+float64(c.B))/2 }
	for y := image.Bounds().Dy() / 2; y < image.Bounds().Dy(); y++ {
		for x := 0; x < image.Bounds().Dx()/2; x++ {
			if c := image.RGBAAt(x, y); greenest(c) > best {
				best, green = greenest(c), c
			}
		}
	}
	if greenest(green) < 8 {
		t.Fatalf("the lower left has no green in it: best %v (%+.1f)", green, best)
	}
	t.Logf("top=%v centre=%v green=%v (%+.1f) worst step=%d at %v", top, middle, green, best, worst, at)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
