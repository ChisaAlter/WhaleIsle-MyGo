package chrome

import (
	"bytes"
	"context"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

func TestAmbientGradientFastSamplesMatchSource(t *testing.T) {
	p := wallpapers.Defaults()
	for _, variant := range []string{"orbs", "aurora", "chaos", "rays"} {
		p.Variant = variant
		c := gradientConfiguration(p, 1440, 864, false, false)
		for _, phase := range []float64{0, 1.25, 7.5, 19.75, 37.125} {
			for _, blob := range c.blobs(phase) {
				fast := prepareGradientBlob(blob)
				for y := -100.0; y < 1000; y += 79 {
					for x := -100.0; x < 1700; x += 83 {
						if a, b := fast.sample(x, y), blob.at(x, y); math.Abs(a.a-b.a) > 1e-10 {
							t.Fatalf("%s phase%v fast=%v source=%v", variant, phase, a, b)
						}
					}
				}
			}
		}
	}
}
func TestAmbientGradientMaskExactAndSampleLifecycle(t *testing.T) {
	p := wallpapers.Defaults()
	p.Effect = "gradient"
	p.EffectExplicit = true
	p.Count = 1
	for i := range p.Colors {
		p.Colors[i] = "#ffffff"
	}
	for _, dark := range []bool{false, true} {
		c := gradientConfiguration(p, 1440, 864, dark, true)
		pixels := renderGradient(context.Background(), c, 0)
		want := uint8(194)
		if dark {
			want = 127
		}
		for i := 0; i < len(pixels.Pix); i += 4 {
			if pixels.Pix[i] != want || pixels.Pix[i+1] != want || pixels.Pix[i+2] != want || pixels.Pix[i+3] != 255 {
				t.Fatalf("dark=%v maskpixel=%v want%d", dark, pixels.Pix[i:i+4], want)
			}
		}
	}
	cache := gradientFrameCache{}
	c := gradientConfiguration(p, 1440, 864, false, false)
	a := cache.get(c, time.Unix(100, 0))
	generation := cache.generation
	c.colors[2] = gradientColor{1, 0, 0, 1}
	b := cache.get(c, time.Unix(101, 0))
	if a == b || cache.generation <= generation {
		t.Fatal("source edit reused previous source")
	}
	cache.reset()
	if cache.frame != nil {
		t.Fatal("reset retained generation")
	}
}
func TestAmbientGradientSurfaceUsesCanvasOnce(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	var phase int
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter("", prefs, true)
		ui.Box(c).Fill().Draw(paint).Children(func() {
			if phase > 0 {
				ui.Box(c).Fill().Draw(Glass(ui.RGB(255, 255, 255), MaterialCanvas)).Children(func() { ui.Box(c).Fill().Draw(Glass(ui.RGB(255, 255, 255), MaterialSidebar)) })
			}
		})
	}, 320, 220)
	before := bytes.Clone(tt.Image().Pix)
	phase = 1
	tt.Frame()
	after := tt.Image()
	for _, pt := range [][2]int{{40, 40}, {160, 110}, {280, 180}} {
		pos := (pt[1]*320 + pt[0]) * 4
		for ch := range 3 {
			want := 255*.45 + float64(before[pos+ch])*.55
			got := after.RGBAAt(pt[0], pt[1])
			v := ([3]uint8{got.R, got.G, got.B})[ch]
			if math.Abs(float64(v)-want) > 2 {
				t.Fatalf("hidden sidebar replaced/doubled canvas at%v channel%d got%d want%.2f", pt, ch, v, want)
			}
		}
	}
}

// A direct two-dimensional reference convolution checks the optimized separable
// loops, including zero extension and normalization, independently of its loops.
func TestAmbientGradientGaussianScalarOracle(t *testing.T) {
	const w, h = 9, 7
	const sigmaX, sigmaY = 1.7, 1.1
	source := make([]float32, w*h*4)
	for i := range source {
		source[i] = float32((i*17)%101) / 101
	}
	original := append([]float32(nil), source...)
	if !gradientGaussian(context.Background(), source, make([]float32, len(source)), w, h, sigmaX, sigmaY) {
		t.Fatal("cancelled")
	}
	weight := func(d int, sigma float64) float64 { return math.Exp(-float64(d*d) / (2 * sigma * sigma)) }
	rx, ry := int(math.Ceil(3*sigmaX)), int(math.Ceil(3*sigmaY))
	normX, normY := 0.0, 0.0
	for d := -rx; d <= rx; d++ {
		normX += weight(d, sigmaX)
	}
	for d := -ry; d <= ry; d++ {
		normY += weight(d, sigmaY)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for channel := 0; channel < 4; channel++ {
				want := 0.0
				for dy := -ry; dy <= ry; dy++ {
					for dx := -rx; dx <= rx; dx++ {
						sx, sy := x+dx, y+dy
						if sx >= 0 && sy >= 0 && sx < w && sy < h {
							want += float64(original[(sy*w+sx)*4+channel]) * weight(dx, sigmaX) * weight(dy, sigmaY) / (normX * normY)
						}
					}
				}
				if got := float64(source[(y*w+x)*4+channel]); math.Abs(got-want) > 3e-7 {
					t.Fatalf("convolution @%d,%d/%d got%.9f scalar%.9f", x, y, channel, got, want)
				}
			}
		}
	}
}

func BenchmarkAmbientGradientCacheHit(b *testing.B) {
	p := wallpapers.Defaults()
	c := gradientConfiguration(p, 1440, 864, false, false)
	cache := gradientFrameCache{}
	now := time.Unix(100, 0)
	cache.get(c, now)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		cache.get(c, now)
	}
}
func BenchmarkAmbientGradientSettingsPanel(b *testing.B) {
	p := wallpapers.Defaults()
	c := gradientConfiguration(p, 1440, 864, false, false)
	f := &gradientFrame{pixels: renderGradient(context.Background(), c, 0), config: c}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		gradientPanelPixels(f, ui.RGB(255, 255, 255), p, false)
	}
}
