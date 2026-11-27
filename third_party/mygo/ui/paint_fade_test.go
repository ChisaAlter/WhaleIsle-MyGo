package ui

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu"
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

// TestFadeHorizontalSingleDrawPassAndBatch verifies that FadeHorizontal paints
// its subtree and text exactly once under a single horizontal-fade clip rather
// than slicing the edges into ~25 1px clip strips and fragmenting GPU batches.
func TestFadeHorizontalSingleDrawPassAndBatch(t *testing.T) {
	drawCalls := 0
	tt := coreNewTester(func(c *context) {
		coreBox(c).Absolute().Left(8).Top(8).Size(120, 28).FadeHorizontal(24, 24).Draw(func(p *Painter, r Rect) {
			drawCalls++
		}).Children(func() {
			coreText(c, "Long sidebar session title that fades smoothly").NoWrap()
		})
	}, 160, 48)
	tt.SetScale(2)
	tt.Frame()
	drawCalls = 0
	tt.Frame()

	if drawCalls != 1 {
		t.Fatalf("FadeHorizontal invoked subtree draw %d times; want 1", drawCalls)
	}
	glyphOps := 0
	fadeClips := 0
	for _, op := range tt.rt.scene.Ops {
		switch op.Kind {
		case scene.OpGlyphs:
			glyphOps++
		case scene.OpPushClip:
			if op.Fade[0] > 0 || op.Fade[1] > 0 {
				fadeClips++
			}
		}
	}
	if fadeClips != 1 {
		t.Fatalf("expected 1 horizontal fade clip op, got %d", fadeClips)
	}
	if glyphOps != 1 {
		t.Fatalf("expected text to be emitted in 1 OpGlyphs pass, got %d", glyphOps)
	}

	var b gpu.Builder
	b.Build(&tt.rt.scene, func(*scene.Image) uintptr { return 1 })
	if len(b.Batches) > 2 {
		t.Fatalf("FadeHorizontal fragmented GPU batches: got %d batches (want <=2)", len(b.Batches))
	}
}

func TestPainterClipFade(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run("scale-"+floatLabel(scale), func(t *testing.T) {
			var clipAfter Rect
			tt := coreNewTester(func(c *context) {
				fadeTheme := *c.Theme()
				fadeTheme.Background = Transparent
				c.SetTheme(&fadeTheme)
				coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
					before := p.ClipBounds()
					p.ClipFade(Rect{X: 8, Y: 4, W: 44, H: 16}, 8, 12, func() {
						p.Fill(Rect{X: 0, Y: 0, W: 64, H: 24}, RGB(220, 40, 20), 0)
					})
					clipAfter = p.ClipBounds()
					if clipAfter != before {
						t.Errorf("ClipFade did not restore p.clip: got %v want %v", clipAfter, before)
					}
					// Drawing after ClipFade outside its rect must remain unclipped.
					p.Fill(Rect{X: 0, Y: 20, W: 4, H: 4}, RGB(20, 180, 60), 0)
				})
			}, 64, 24)
			tt.SetScale(scale)
			tt.Frame()
			frame := tt.Image()
			x0, x1 := int(round(8*scale)), int(round(52*scale))
			y := int(round(10 * scale))
			fl, fr := 8*float64(scale), 12*float64(scale)
			for x := x0; x < x1; x++ {
				expect := math.Min(1, math.Min((float64(x-x0)+.5)/fl, (float64(x1-x)-.5)/fr))
				got := frame.RGBAAt(x, y)
				if math.Abs(float64(got.A)-expect*255) > 2 {
					t.Fatalf("ClipFade alpha x=%d got=%d want=%.1f", x, got.A, expect*255)
				}
			}
			if frame.RGBAAt(x0-1, y).A != 0 || frame.RGBAAt(x1, y).A != 0 {
				t.Fatalf("ClipFade leaked outside rect: left=%v right=%v", frame.RGBAAt(x0-1, y), frame.RGBAAt(x1, y))
			}
			// Verify post-ClipFade drawing at (1, 21) succeeded.
			if got := frame.RGBAAt(int(round(1*scale)), int(round(21*scale))); got.G < 160 || got.A != 255 {
				t.Fatalf("drawing after ClipFade was clipped: %+v", got)
			}
		})
	}
}

func TestFadeOverflowRightShortVsLongAndZeroShift(t *testing.T) {
	const (
		shortTitle = "Documents"
		longTitle  = "Documents and workspace directory with a very long title that overflows the action slot"
		insetR     = float32(48)
		fadeR      = float32(14)
	)
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run("scale-"+floatLabel(scale), func(t *testing.T) {
			fading := false
			tt := coreNewTester(func(c *context) {
				fadeTheme := *c.Theme()
				fadeTheme.Background = Transparent
				c.SetTheme(&fadeTheme)
				coreColumn(c).Width(180).Gap(6).Padding(4).Children(func() {
					s := coreText(c, shortTitle).FillWidth().SingleLine().FontSize(13).TextColor(RGB(240, 240, 240)).Label("short")
					l := coreText(c, longTitle).FillWidth().SingleLine().FontSize(13).TextColor(RGB(240, 240, 240)).Label("long-single")
					n := coreText(c, longTitle).FillWidth().NoWrap().FontSize(13).TextColor(RGB(240, 240, 240)).Label("long-nowrap")
					if fading {
						s.FadeOverflowRight(insetR, fadeR)
						l.FadeOverflowRight(insetR, fadeR)
						n.FadeOverflowRight(insetR, fadeR)
					}
				})
			}, 200, 96)
			tt.SetScale(scale)
			tt.Frame()

			baseFlow := tt.FlowSnapshot()
			plain := image.NewRGBA(tt.Image().Bounds())
			copy(plain.Pix, tt.Image().Pix)
			shortRect, _ := tt.Find("short")
			longSingleRect, _ := tt.Find("long-single")
			longNowrapRect, _ := tt.Find("long-nowrap")

			fading = true
			tt.Frame()
			fadedFlow := tt.FlowSnapshot()
			faded := tt.Image()

			// 1. FlowSnapshot geometry must have dx=dy=dw=dh=0 for every node.
			if len(baseFlow) != len(fadedFlow) {
				t.Fatalf("FlowSnapshot node count changed: %d -> %d", len(baseFlow), len(fadedFlow))
			}
			for i := range baseFlow {
				b, g := baseFlow[i], fadedFlow[i]
				if b.ID != g.ID || b.Rect != g.Rect {
					t.Fatalf("FadeOverflowRight shifted node %d: base=%v got=%v", i, b, g)
				}
			}

			// 2. Short text ("Documents") must remain byte-for-byte crisp with no fade.
			sx0, sy0 := int(round(shortRect.X*scale)), int(round(shortRect.Y*scale))
			sx1, sy1 := int(round((shortRect.X+shortRect.W)*scale)), int(round((shortRect.Y+shortRect.H)*scale))
			shortInk := 0
			for y := sy0; y < sy1; y++ {
				for x := sx0; x < sx1; x++ {
					p0, p1 := plain.RGBAAt(x, y), faded.RGBAAt(x, y)
					if p0 != p1 {
						t.Fatalf("short text changed at (%d,%d): plain=%v faded=%v", x, y, p0, p1)
					}
					if p1.A > 100 {
						shortInk++
					}
				}
			}
			if shortInk < 10 {
				t.Fatalf("expected visible short text ink, got %d pixels", shortInk)
			}

			// 3. Long texts (SingleLine & NoWrap) must hard-clip to alpha==0 at >= right
			//    and smoothly attenuate alpha inside [right-fadeR, right).
			for _, tc := range []struct {
				name string
				r    Rect
			}{
				{"long-single", longSingleRect},
				{"long-nowrap", longNowrapRect},
			} {
				clipRight := int(round((tc.r.X + tc.r.W - insetR) * scale))
				boxRight := int(round((tc.r.X + tc.r.W) * scale))
				fadeStart := int(math.Floor(float64(clipRight) - float64(fadeR)*float64(scale)))
				y0, y1 := int(round(tc.r.Y*scale)), int(round((tc.r.Y+tc.r.H)*scale))

				for y := y0; y < y1; y++ {
					for x := clipRight; x < boxRight; x++ {
						if got := faded.RGBAAt(x, y); got.A != 0 {
							t.Fatalf("%s leaked past right clip at (%d,%d) (clipRight=%d): %v", tc.name, x, y, clipRight, got)
						}
					}
				}
				attenuated := 0
				for y := y0; y < y1; y++ {
					for x := fadeStart; x < clipRight; x++ {
						p0, p1 := plain.RGBAAt(x, y), faded.RGBAAt(x, y)
						if p0.A > 80 && p1.A < p0.A {
							attenuated++
						}
					}
				}
				if attenuated < 5 {
					t.Fatalf("%s did not smoothly fade in [%d,%d): only %d attenuated pixels", tc.name, fadeStart, clipRight, attenuated)
				}
			}
		})
	}
}

func TestFadeOverflowRightPublicFacadeZeroShiftOnHover(t *testing.T) {
	tt := NewTester(func(c *Context) {
		row := Row(c).Width(180).Height(32).PaddingX(8).Label("ws-row")
		hovered := row.Hovered() || row.FocusWithin()
		row.Children(func() {
			txt := Text(c, "A very long workspace directory title that collides with hover buttons").Grow(1).SingleLine()
			if hovered {
				txt.FadeOverflowRight(45, 14)
			}
		})
	}, 220, 48)
	tt.SetScale(1.5)
	AssertHoverFocusZeroShift(t, tt)
}
