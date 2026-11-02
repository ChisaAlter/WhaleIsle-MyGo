package ui

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

func TestFadeHorizontalAlphaAndDisjointClips(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run("scale-"+floatLabel(scale), func(t *testing.T) {
			tt := coreNewTester(func(c *context) {
				fadeTheme := *c.Theme()
				fadeTheme.Background = Transparent
				c.SetTheme(&fadeTheme)
				coreBox(c).Absolute().Left(5).Top(4).Size(40, 12).FadeHorizontal(8, 8).Children(func() {
					coreBox(c).Fill().Background(RGB(220, 40, 20))
				})
			}, 60, 24)
			tt.SetScale(scale)
			tt.Frame()
			frame := tt.Image()
			x0, x1 := int(round(5*scale)), int(round(45*scale))
			y := int(round(8 * scale))
			for x := x0; x < x1; x++ {
				expect := math.Min(1, math.Min((float64(x-x0)+.5)/(8*float64(scale)), (float64(x1-x)-.5)/(8*float64(scale))))
				got := frame.RGBAAt(x, y)
				if math.Abs(float64(got.A)-expect*255) > 2 {
					t.Fatalf("alpha x=%d got=%d want=%f", x, got.A, expect*255)
				}
				if got.A > 0 && math.Abs(float64(got.R)-float64(got.A)*220/255) > 1.5 {
					t.Fatalf("fade changed foreground hue: %+v", got)
				}
			}
			if frame.RGBAAt(x0-1, y).A != 0 || frame.RGBAAt(x1, y).A != 0 {
				t.Fatal("mask painted beyond horizontal element bounds")
			}
			// The center is drawn once; no output column is blended twice.
			counts := make([]int, frame.Bounds().Dx())
			var stack []scene.Rect
			clip := scene.Rect{W: float32(frame.Bounds().Dx()), H: float32(frame.Bounds().Dy())}
			for _, op := range tt.rt.scene.Ops {
				switch op.Kind {
				case scene.OpPushClip:
					stack = append(stack, clip)
					clip = clip.Intersect(op.Rect)
				case scene.OpPopClip:
					clip = stack[len(stack)-1]
					stack = stack[:len(stack)-1]
				case scene.OpFill:
					if op.Color.R != 220 || op.Color.G != 40 {
						continue
					}
					box := op.Rect.Intersect(clip)
					for x := int(box.X); x < int(box.X+box.W); x++ {
						counts[x]++
					}
				}
			}
			for x := x0; x < x1; x++ {
				if counts[x] != 1 {
					t.Fatalf("column %d painted %d times", x, counts[x])
				}
			}
		})
	}
}

func TestFadeHorizontalRetainsCheckerboardAndColorImage(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		pixels := image.NewRGBA(image.Rect(0, 0, 40, 12))
		for y := 0; y < 12; y++ {
			for x := 0; x < 40; x++ {
				pixels.SetRGBA(x, y, color.RGBA{R: 210, G: 60, B: 20, A: 255})
			}
		}
		bitmap := NewBitmap(pixels)
		fade := true
		tt := coreNewTester(func(c *context) {
			coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
				for y := float32(0); y < r.H; y += 4 {
					for x := float32(0); x < r.W; x += 4 {
						bg := RGB(20, 90, 170)
						if (int(x/4)+int(y/4))%2 == 0 {
							bg = RGB(180, 210, 30)
						}
						p.Fill(Rect{X: x, Y: y, W: 4, H: 4}, bg, 0)
					}
				}
			})
			box := coreBox(c).Absolute().Left(4).Top(4).Size(40, 12)
			if fade {
				box.FadeHorizontal(8, 8)
			}
			box.Children(func() { coreImage(c, bitmap).Fill() })
		}, 56, 24)
		tt.SetScale(scale)
		tt.Frame()
		masked := image.NewRGBA(tt.Image().Bounds())
		copy(masked.Pix, tt.Image().Pix)
		fade = false
		tt.Frame()
		plain := tt.Image()
		x0 := int(round(4 * scale))
		y := int(round(8 * scale))
		x := x0 + 1
		if masked.RGBAAt(x, y) == plain.RGBAAt(x, y) {
			t.Fatal("colored image edge did not fade")
		}
		if masked.RGBAAt(x0+int(20*scale), y) != plain.RGBAAt(x0+int(20*scale), y) {
			t.Fatal("center image changed")
		}
		if masked.RGBAAt(x0-1, y) != plain.RGBAAt(x0-1, y) {
			t.Fatal("checkerboard outside mask changed")
		}
	}
}

func TestFadeHorizontalValidationOverlapAndHits(t *testing.T) {
	clicks := 0
	tt := coreNewTester(func(c *context) {
		fadeTheme := *c.Theme()
		fadeTheme.Background = Transparent
		c.SetTheme(&fadeTheme)
		if coreBox(c).Size(10, 12).FadeHorizontal(20, 20).Background(RGB(220, 40, 20)).Label("faded").Clicked() {
			clicks++
		}
	}, 30, 24)
	tt.ClickAt(.5, 6)
	if clicks != 1 {
		t.Fatal("fade changed pointer bounds")
	}
	if got := tt.Image().RGBAAt(4, 6); got.A < 225 || got.A > 232 {
		t.Fatalf("overlap was painted twice or erased: %+v", got)
	}
	var invalid [2]float32
	coreNewTester(func(c *context) {
		e := coreBox(c).FadeHorizontal(float32(math.NaN()), float32(math.Inf(1)))
		invalid = e.fade
	}, 20, 20)
	if invalid != [2]float32{} {
		t.Fatalf("nonfinite fade retained: %v", invalid)
	}
}

func TestFadeHorizontalNativeColorEmoji(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("native color emoji availability varies")
	}
	fade := false
	tt := coreNewTester(func(c *context) {
		fadeTheme := *c.Theme()
		fadeTheme.Background = Transparent
		c.SetTheme(&fadeTheme)
		lane := coreBox(c).Absolute().Left(4).Top(4).Size(64, 44)
		if fade {
			lane.FadeHorizontal(32, 32)
		}
		lane.Children(func() { coreText(c, "🎉").FontSize(32).TextColor(RGB(255, 255, 255)).NoWrap() })
	}, 76, 56)
	tt.SetScale(1.5)
	tt.Frame()
	plain := image.NewRGBA(tt.Image().Bounds())
	copy(plain.Pix, tt.Image().Pix)
	colored := false
	for _, g := range tt.rt.scene.Glyphs {
		colored = colored || g.Colored
	}
	if !colored {
		t.Fatal("native emoji was not a color glyph")
	}
	fade = true
	tt.Frame()
	changed := 0
	for y := 6; y < 72; y++ {
		for x := 6; x < 102; x++ {
			old, got := plain.RGBAAt(x, y), tt.Image().RGBAAt(x, y)
			if old.A < 100 {
				continue
			}
			alpha := math.Min(1, math.Min((float64(x-6)+.5)/48, (float64(102-x)-.5)/48))
			if math.Abs(float64(got.A)-float64(old.A)*alpha) > 3 {
				t.Fatalf("emoji alpha x=%d got=%d original=%d multiplier=%f", x, got.A, old.A, alpha)
			}
			if got.A < old.A {
				changed++
			}
		}
	}
	if changed < 20 {
		t.Fatalf("emoji fade missing: %d pixels", changed)
	}
}

func floatLabel(n float32) string {
	if n == 1.5 {
		return "1.5"
	}
	if n == 2 {
		return "2"
	}
	return "1"
}
