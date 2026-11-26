package glass

import (
	"image"
	"testing"

	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/ui"
)

func ambientTestColors() (ui.Color, ui.Color, [5]ui.Color) {
	return ui.Hex("#283142"), ui.Hex("#172554"), [5]ui.Color{
		ui.Hex("#34d399"),
		ui.Hex("#22d3ee"),
		ui.Hex("#a78bfa"),
		ui.Hex("#4ade80"),
		ui.Hex("#38bdf8"),
	}
}

func renderAmbientTestImage(b AmbientBackdrop, w, h int) *image.RGBA {
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Material(b)
	}
	return ui.NewTester(view, w, h).Image()
}

func imagePixelDiff(a, b *image.RGBA) (changed int, totalDiff int) {
	bounds := a.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pa := a.RGBAAt(x, y)
			pb := b.RGBAAt(x, y)
			d := absInt(int(pa.R)-int(pb.R)) + absInt(int(pa.G)-int(pb.G)) + absInt(int(pa.B)-int(pb.B))
			if d > 3 {
				changed++
			}
			totalDiff += d
		}
	}
	return changed, totalDiff
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ambientScene returns a 320×160 scene covering all four AmbientBackdrop
// variants, clipped/rounded regions, opacity, dark mode, and dithered blur
// so GPU renderers can be compared pixel-by-pixel against the CPU renderer.
func ambientScene() *scene.Scene {
	s := &scene.Scene{Width: 320, Height: 160, Clear: scene.Color{R: 16, G: 20, B: 28, A: 255}}
	baseFrom, baseTo, colors := ambientTestColors()
	addAmbient := func(r scene.Rect, radii [4]float32, v AmbientVariant, count int, sec float32, dark bool, op float32) {
		var dk float32
		if dark {
			dk = 1
		}
		params := [5][4]float32{
			{packRGB(baseFrom), packRGB(baseTo), packRGB(colors[0]), packRGB(colors[1])},
			{packRGB(colors[2]), packRGB(colors[3]), packRGB(colors[4]), float32(v)},
			{float32(count), sec, dk, op},
			{1, 0, 0, 0},
			{},
		}
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpEffect, Rect: r, Radii: radii, Start: int32(len(s.Effects))})
		s.Effects = append(s.Effects, scene.EffectOp{Effect: ambientEffect, Params: params})
	}
	addAmbient(scene.Rect{X: 0, Y: 0, W: 160, H: 80}, r4(0), AmbientOrbs, 5, 2.5, true, 1)
	addAmbient(scene.Rect{X: 160, Y: 0, W: 160, H: 80}, r4(10), AmbientAurora, 5, 4.0, false, 1)
	addAmbient(scene.Rect{X: 0, Y: 80, W: 160, H: 80}, r4(8), AmbientChaos, 5, 3.0, true, 0.85)
	addAmbient(scene.Rect{X: 160, Y: 80, W: 160, H: 80}, r4(0), AmbientRays, 5, 6.0, true, 1)

	p2, p3 := blurTone{dither: true}.params()
	s.Ops = append(s.Ops, scene.Op{Kind: scene.OpEffect, Rect: scene.Rect{X: 20, Y: 20, W: 120, H: 40}, Radii: r4(8), Start: int32(len(s.Effects))})
	s.Effects = append(s.Effects, scene.EffectOp{Effect: blurEffect, Blur: 6, Params: [5][4]float32{{}, {6, 6, -1, 0}, p2, p3}})
	return s
}

func TestAmbientPackRoundTrip(t *testing.T) {
	cases := []ui.Color{
		ui.RGB(0, 0, 0),
		ui.RGB(255, 255, 255),
		ui.RGB(255, 0, 1),
		ui.RGB(0, 255, 0),
		ui.RGB(1, 255, 254),
		ui.Hex("#34d399"),
		ui.Hex("#283142"),
	}
	for _, c := range cases {
		got := unpackRGB(packRGB(c))
		want := [3]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255}
		for i := range 3 {
			if abs(got[i]-want[i]) > 1e-6 {
				t.Fatalf("color %v channel %d: got %v, want %v", c, i, got[i], want[i])
			}
		}
	}
}

func TestAmbientVariantsDistinct(t *testing.T) {
	baseFrom, baseTo, colors := ambientTestColors()
	variants := []AmbientVariant{AmbientOrbs, AmbientAurora, AmbientChaos, AmbientRays}
	imgs := make([]*image.RGBA, len(variants))
	for i, v := range variants {
		imgs[i] = renderAmbientTestImage(AmbientBackdrop{
			Variant:  v,
			BaseFrom: baseFrom,
			BaseTo:   baseTo,
			Colors:   colors,
			Count:    5,
			Time:     3.5,
			Dark:     true,
		}, 160, 120)
	}
	for i := 0; i < len(variants); i++ {
		for j := i + 1; j < len(variants); j++ {
			changed, total := imagePixelDiff(imgs[i], imgs[j])
			if changed < 500 || total < 10000 {
				t.Errorf("variants %d and %d too similar: changed=%d totalDiff=%d", variants[i], variants[j], changed, total)
			}
		}
	}
}

func TestAmbientTimeAnimation(t *testing.T) {
	baseFrom, baseTo, colors := ambientTestColors()
	for _, v := range []AmbientVariant{AmbientOrbs, AmbientAurora, AmbientChaos, AmbientRays} {
		t0 := renderAmbientTestImage(AmbientBackdrop{
			Variant: v, BaseFrom: baseFrom, BaseTo: baseTo, Colors: colors, Count: 5, Time: 0, Dark: true,
		}, 160, 120)
		t6 := renderAmbientTestImage(AmbientBackdrop{
			Variant: v, BaseFrom: baseFrom, BaseTo: baseTo, Colors: colors, Count: 5, Time: 6, Dark: true,
		}, 160, 120)
		changed, total := imagePixelDiff(t0, t6)
		if changed < 200 || total < 3000 {
			t.Errorf("variant %d did not animate visibly between Time=0 and Time=6: changed=%d totalDiff=%d", v, changed, total)
		}
	}
}

func TestAmbientCountDifference(t *testing.T) {
	baseFrom, baseTo, colors := ambientTestColors()
	c1 := renderAmbientTestImage(AmbientBackdrop{
		Variant: AmbientOrbs, BaseFrom: baseFrom, BaseTo: baseTo, Colors: colors, Count: 1, Time: 2, Dark: true,
	}, 160, 120)
	c5 := renderAmbientTestImage(AmbientBackdrop{
		Variant: AmbientOrbs, BaseFrom: baseFrom, BaseTo: baseTo, Colors: colors, Count: 5, Time: 2, Dark: true,
	}, 160, 120)
	changed, total := imagePixelDiff(c1, c5)
	if changed < 500 || total < 8000 {
		t.Errorf("Count=1 and Count=5 too similar: changed=%d totalDiff=%d", changed, total)
	}
}

func TestAmbientOpacityAndBackdropFlag(t *testing.T) {
	if ambientEffect.Backdrop {
		t.Fatal("ambientEffect.Backdrop must be false for zero backbuffer copy")
	}
	baseFrom, baseTo, colors := ambientTestColors()
	view := func(op float32) func(*ui.Context) {
		return func(c *ui.Context) {
			ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
				ui.Box(c).Fill().Material(AmbientBackdrop{
					Variant: AmbientOrbs, BaseFrom: baseFrom, BaseTo: baseTo, Colors: colors, Count: 3, Time: 1, Opacity: op,
				})
			})
		}
	}
	full := ui.NewTester(view(0), 80, 60).Image()
	half := ui.NewTester(view(0.5), 80, 60).Image()
	changed, _ := imagePixelDiff(full, half)
	if changed < 1000 {
		t.Errorf("Opacity=0.5 should differ from default Opacity<=0 (1.0): changed=%d", changed)
	}
	if s := (AmbientBackdrop{Variant: AmbientAurora, Count: 4}).String(); s != "ambient(aurora, count=4)" {
		t.Errorf("String() = %q", s)
	}
}

func TestBlurDitherOption(t *testing.T) {
	view := func(dither bool) func(*ui.Context) {
		return func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
				p.FillGradient(r, ui.LinearGradient{From: ui.RGB(20, 24, 34), To: ui.RGB(36, 44, 62), Angle: 180}, 0)
			}).Children(func() {
				ui.Box(c).Absolute().Left(0).Top(0).Size(120, 80).Material(Blur{Radius: 8, Dither: dither})
			})
		}
	}
	noDither := ui.NewTester(view(false), 120, 80).Image()
	withDither := ui.NewTester(view(true), 120, 80).Image()
	maxDelta := 0
	diffPixels := 0
	for y := 10; y < 70; y++ {
		for x := 10; x < 110; x++ {
			a := noDither.RGBAAt(x, y)
			b := withDither.RGBAAt(x, y)
			for _, d := range []int{absInt(int(a.R) - int(b.R)), absInt(int(a.G) - int(b.G)), absInt(int(a.B) - int(b.B))} {
				if d > 0 {
					diffPixels++
				}
				if d > maxDelta {
					maxDelta = d
				}
			}
		}
	}
	if diffPixels == 0 || maxDelta > 1 {
		t.Errorf("dither should introduce subtle ±1 LSB variation: diffPixels=%d maxDelta=%d", diffPixels, maxDelta)
	}

	// Verify blurTone with Dither=false keeps p3.w == 0 even when Saturation != 0.
	_, p3NoDither := blurTone{saturation: 0.5}.params()
	if p3NoDither[3] != 0 {
		t.Errorf("blurTone without dither set p3.w = %v, want 0", p3NoDither[3])
	}
	_, p3WithDither := blurTone{saturation: 0.5, dither: true}.params()
	if p3WithDither[3] != 1 {
		t.Errorf("blurTone with dither set p3.w = %v, want 1", p3WithDither[3])
	}

	// Verify raster rendering of ambientScene completes with opaque corners where expected.
	s := ambientScene()
	img := raster.NewImage(s.Width, s.Height)
	raster.Render(img, s)
	p := img.Pix[10*img.Stride+4*10:]
	if p[3] != 255 {
		t.Errorf("ambientScene pixel alpha = %d, want 255", p[3])
	}
}
