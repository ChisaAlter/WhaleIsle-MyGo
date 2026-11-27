package chrome

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

func testChannelSpan(tt *ui.Tester, y, x0, x1 int) int {
	img := tt.Image()
	loR, hiR := 255, 0
	loB, hiB := 255, 0
	for x := x0; x < x1; x++ {
		p := img.RGBAAt(x, y)
		loR, hiR = min(loR, int(p.R)), max(hiR, int(p.R))
		loB, hiB = min(loB, int(p.B)), max(hiB, int(p.B))
	}
	return max(hiR-loR, hiB-loB)
}

func TestGradientModeDrawGlassBlursBackdropAndDialogs(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	prefs.Glass = 60
	prefs.Terminal = 55

	paintStripes := func(p *ui.Painter, r ui.Rect) {
		c1 := ui.RGB(20, 120, 240)
		c2 := ui.RGB(240, 120, 20)
		for i, x := 0, float32(0); x < r.W; i, x = i+1, x+4 {
			col := c1
			if i%2 == 1 {
				col = c2
			}
			p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 4, H: r.H}, col, 0)
		}
	}

	for _, mat := range []Material{MaterialMenu, MaterialTerminal} {
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.DarkTheme())
			paint := WallpaperBackgroundPainter("", prefs, true)
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
				paint(p, r)
				paintStripes(p, r)
			}).Children(func() {
				ui.Box(c).Absolute().Left(30).Top(20).Width(120).Height(80).
					Draw(Glass(ui.RGB(48, 49, 54), mat, 8))
			})
		}, 200, 140)
		tt.Frame()
		rawSpan := testChannelSpan(tt, 10, 10, 190)
		if rawSpan < 150 {
			t.Fatalf("expected vivid test stripes outside glass, got span=%d", rawSpan)
		}
		blurredSpan := testChannelSpan(tt, 60, 50, 130)
		if blurredSpan > rawSpan/3 {
			t.Fatalf("material %d in gradient mode did not blur backdrop: blurredSpan=%d rawSpan=%d", mat, blurredSpan, rawSpan)
		}
	}

	// Verify DialogSurfaceTheme and DialogDecorTheme in gradient mode use
	// modalPanelGlass (26px dark blur), deep charcoal tint, 0.5 DIP dark outer
	// bevel border, and bright inner 1px specular rim + top highlight at y+1.
	for _, kind := range []string{"surface", "decor"} {
		t.Run(kind, func(t *testing.T) {
			open := true
			var dlgRect ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(ui.DarkTheme())
				paint := WallpaperBackgroundPainter("", prefs, true)
				ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
					paint(p, r)
					paintStripes(p, r)
				}).Children(func() {
					ui.DialogBase(c.Key("dlg"), &open, func(backdrop, panel ui.Element) {
						if kind == "surface" {
							DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 160, 12)
						} else {
							panel.Width(160).Height(90)
							DialogDecorTheme(c, backdrop, panel, c.Theme(), 16)
						}
						panel.Children(func() {
							ui.Box(c).Size(120, 60)
						})
						dlgRect = panel.Bounds()
					})
				})
			}, 260, 180)
			tt.Frame()
			tt.Frame()
			img := tt.Image()
			x0, x1 := int(dlgRect.X)+24, int(dlgRect.X+dlgRect.W)-24
			midY := int(dlgRect.Y + dlgRect.H/2)
			blurredSpan := testChannelSpan(tt, midY, x0, x1)
			if blurredSpan > 40 {
				t.Fatalf("dialog %s in gradient mode did not blur stripes: span=%d", kind, blurredSpan)
			}
			midX := int(dlgRect.X + dlgRect.W/2)
			topY := int(math.Round(float64(dlgRect.Y)))
			outerEdge := img.RGBAAt(midX, topY)
			innerEdge := img.RGBAAt(midX, topY+1)
			interior := img.RGBAAt(midX, topY+14)
			outerLum := RelativeLuminance(pixel(outerEdge))
			edgeLum := RelativeLuminance(pixel(innerEdge))
			intLum := RelativeLuminance(pixel(interior))
			if outerLum >= intLum {
				t.Fatalf("dialog %s missing dark 0.5 DIP outer bevel border: outer=%v interior=%v", kind, outerEdge, interior)
			}
			if edgeLum <= intLum+0.02 {
				t.Fatalf("dialog %s missing bright inner specular rim + top highlight: edge=%v interior=%v", kind, innerEdge, interior)
			}
			_ = color.RGBA{}
		})
	}
}

func TestWallpaperBlurPreservesEdgesWithoutBleed(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 220, G: 220, B: 220, A: 255}, 64, 64)
	for _, pix := range []float64{0, 100} {
		prefs := wallpapers.Defaults()
		prefs.Effect = "none"
		prefs.Blur = 100
		prefs.Pixelate = pix
		const w, h = 320, 200
		tt := ui.NewTester(func(c *ui.Context) {
			paint := WallpaperBackgroundPainter(url, prefs, true)
			ui.Box(c).Fill().Draw(paint)
		}, w, h)
		tt.Frame()
		img := tt.Image()
		center := img.RGBAAt(w/2, h/2)
		for _, pt := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}, {w / 2, 0}, {0, h / 2}} {
			edge := img.RGBAAt(pt[0], pt[1])
			if absInt(int(edge.R)-int(center.R)) > 3 || absInt(int(edge.G)-int(center.G)) > 3 || absInt(int(edge.B)-int(center.B)) > 3 {
				t.Fatalf("pixelate=%v edge %v=%v diverged from center %v", pix, pt, edge, center)
			}
		}
	}
}

func TestAcrylicCanvasAlphaAndMenuSolidityCurves(t *testing.T) {
	for _, tc := range []struct {
		glass      float64
		canvas     float32
		menuLight  float32
		menuDark   float32
		modalLight float32
		modalDark  float32
	}{
		{0, 0, 0, 0, 0, 0},
		{40, 0.06, 0.22, 0.24, 0.24, 0.27},
		{60, 0.16, 0.33, 0.36, 0.36, 0.405},
		{80, 0.30, 0.44, 0.48, 0.48, 0.54},
		{100, 1.00, 1.00, 1.00, 1.00, 1.00},
	} {
		p := wallpapers.Defaults()
		p.Glass = tc.glass
		if got := AcrylicCanvasAlpha(p); math.Abs(float64(got-tc.canvas)) > 1e-4 {
			t.Fatalf("AcrylicCanvasAlpha(%.0f) = %v, want %v", tc.glass, got, tc.canvas)
		}
		if got := MenuSolidity(p, false); math.Abs(float64(got-tc.menuLight)) > 1e-4 {
			t.Fatalf("MenuSolidity(%.0f, light) = %v, want %v", tc.glass, got, tc.menuLight)
		}
		if got := MenuSolidity(p, true); math.Abs(float64(got-tc.menuDark)) > 1e-4 {
			t.Fatalf("MenuSolidity(%.0f, dark) = %v, want %v", tc.glass, got, tc.menuDark)
		}
		if got := ModalSolidity(p, false); math.Abs(float64(got-tc.modalLight)) > 1e-4 {
			t.Fatalf("ModalSolidity(%.0f, light) = %v, want %v", tc.glass, got, tc.modalLight)
		}
		if got := ModalSolidity(p, true); math.Abs(float64(got-tc.modalDark)) > 1e-4 {
			t.Fatalf("ModalSolidity(%.0f, dark) = %v, want %v", tc.glass, got, tc.modalDark)
		}
	}
}

func TestMaterialTerminalShowThroughUnderDarkAcrylic(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })

	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Glass = 80
	prefs.Terminal = 50
	ConfigureBackdrop("", prefs)

	termBase := ui.RGB(20, 20, 20)
	desktopBlue := ui.RGB(20, 100, 220)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Box(c).Fill().Background(desktopBlue).Children(func() {
			ui.Box(c).Fill().Draw(Glass(termBase, MaterialTerminal, 0))
		})
	}, 60, 40)
	tt.Frame()
	got := tt.Image().RGBAAt(30, 20)
	wantB := float64(termBase.B)*0.5 + float64(desktopBlue.B)*0.5
	if math.Abs(float64(got.B)-wantB) > 3 {
		t.Fatalf("dark Acrylic terminal pixel = %+v, want blue ~%.0f (50%% show-through)", got, wantB)
	}

	// Verify MenuStyle keeps its material contract unchanged.
	if st := MenuStyle(theme.Resolve(theme.IDDeepseekDark).Active(true), true); st.Material != nil {
		t.Fatalf("MenuStyle without backdrop should stay opaque without material, got %T", st.Material)
	}
}
