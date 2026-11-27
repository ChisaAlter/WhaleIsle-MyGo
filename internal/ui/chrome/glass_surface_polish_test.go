package chrome

import (
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

func TestApplyMenuSurfaceBlursAndFallsBackCleanly(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})

	paintStripes := func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+4 {
				col := ui.RGB(20, 120, 240)
				if i%2 == 1 {
					col = ui.RGB(240, 120, 20)
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 4, H: r.H}, col, 0)
			}
		})
	}

	// Off: keeps solid Popover background with zero stripe leakage.
	ConfigureBackdrop("", wallpapers.Defaults())
	design.SetGlass(false, nil)
	var bOff ui.Rect
	ttOff := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			paintStripes(c)
			bOff = ApplyMenuSurface(ui.Box(c).Absolute().Left(30).Top(20).Width(120).Height(80), design.For(c), 12).Bounds()
		})
	}, 200, 140)
	ttOff.Frame()
	if span := testChannelSpan(ttOff, int(bOff.Y+bOff.H/2), int(bOff.X+20), int(bOff.X+bOff.W-20)); span != 0 {
		t.Fatalf("expected opaque fallback with glass off, got stripe span=%d", span)
	}

	// On: blurs backdrop (20/22 DIP, high saturation) with a crisp 0.5 DIP outer
	// border and a bright inner 1px specular rim + top highlight at y+1.
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Popover = p.Popover.Alpha(0.44)
	})
	var bOn ui.Rect
	ttOn := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			paintStripes(c)
			bOn = ApplyMenuSurface(ui.Box(c).Absolute().Left(30).Top(20).Width(120).Height(80), design.For(c), 12).Bounds()
		})
	}, 200, 140)
	ttOn.Frame()
	raw := testChannelSpan(ttOn, 8, 10, 190)
	blurred := testChannelSpan(ttOn, int(bOn.Y+bOn.H/2), int(bOn.X+20), int(bOn.X+bOn.W-20))
	if blurred > raw/4 {
		t.Fatalf("ApplyMenuSurface did not blur stripes enough: blurred=%d raw=%d", blurred, raw)
	}
	img := ttOn.Image()
	outerLum := RelativeLuminance(pixel(img.RGBAAt(int(bOn.X+bOn.W/2), int(bOn.Y))))
	innerEdgeLum := RelativeLuminance(pixel(img.RGBAAt(int(bOn.X+bOn.W/2), int(bOn.Y+1))))
	inLum := RelativeLuminance(pixel(img.RGBAAt(int(bOn.X+bOn.W/2), int(bOn.Y+10))))
	if outerLum >= inLum {
		t.Fatalf("ApplyMenuSurface missing crisp 0.5 DIP outer border: outer=%.3f inside=%.3f", outerLum, inLum)
	}
	if innerEdgeLum <= inLum+0.02 {
		t.Fatalf("ApplyMenuSurface missing bright inner specular rim + top highlight: innerEdge=%.3f inside=%.3f", innerEdgeLum, inLum)
	}
}

func TestDrawGlassRim150DPIZeroGapOnePhysicalPixel(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Popover = p.Popover.Alpha(0.48)
	})
	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			var box ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				if dark {
					c.SetTheme(ui.DarkTheme())
				}
				bg := ui.RGB(140, 150, 165)
				ui.Box(c).Fill().Background(bg).Children(func() {
					box = ApplyMenuSurface(ui.Box(c).Absolute().Left(30).Top(20).Width(120).Height(80), design.For(c), 12).Bounds()
				})
			}, 200, 140)
			tt.SetScale(1.5)
			tt.Frame()
			img := tt.Image()
			midX := int((box.X + box.W/2) * 1.5)
			topPx := int(box.Y * 1.5)
			outerLum := RelativeLuminance(pixel(img.RGBAAt(midX, topPx)))
			rimLum := RelativeLuminance(pixel(img.RGBAAt(midX, topPx+1)))
			nextLum := RelativeLuminance(pixel(img.RGBAAt(midX, topPx+2)))
			bodyLum := RelativeLuminance(pixel(img.RGBAAt(midX, topPx+16)))
			if outerLum >= bodyLum {
				t.Fatalf("%s 150%% outer border at y=%d not darker than body: outer=%.4f body=%.4f", name, topPx, outerLum, bodyLum)
			}
			if rimLum <= bodyLum+0.02 {
				t.Fatalf("%s 150%% inner specular rim at y=%d missing or gapped: rim=%.4f body=%.4f", name, topPx+1, rimLum, bodyLum)
			}
			if nextLum > bodyLum+0.02 {
				t.Fatalf("%s 150%% inner rim bled into 2nd pixel y=%d: next=%.4f body=%.4f", name, topPx+2, nextLum, bodyLum)
			}
		})
	}
}

func TestDialogMaskDimsWithoutFullscreenBlurUnderGlass(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	url := pngDataURL(t, color.RGBA{R: 200, G: 200, B: 200, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	ConfigureBackdrop(url, prefs)

	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+2 {
				col := ui.RGB(20, 40, 220)
				if i%2 == 1 {
					col = ui.RGB(230, 180, 30)
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 2, H: r.H}, col, 0)
			}
		}).Children(func() {
			ui.DialogBase(c.Key("mask-blur-dlg"), &open, func(backdrop, panel ui.Element) {
				DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 120, 12).Children(func() {
					ui.Box(c).Size(80, 40)
				})
			})
		})
	}, 260, 180)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	// Outside the centered dialog (x=10..50, y=20), pure dim scrim keeps 2px
	// stripes crisp (zero fullscreen blur) while dimming peak brightness.
	if span := testChannelSpan(tt, 20, 10, 50); span < 120 {
		t.Fatalf("expected modal mask to dim without fullscreen blur, got softened span=%d", span)
	}
	var peakR uint8
	img := tt.Image()
	for x := 10; x < 50; x++ {
		if r := img.RGBAAt(x, 20).R; r > peakR {
			peakR = r
		}
	}
	if peakR >= 210 {
		t.Fatalf("expected modal mask dim fill to darken 230-red stripe, got peakR=%d", peakR)
	}
}

func TestDialogPureGlassEnabledBlursAndStaysOpaqueDuringReveal(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	ConfigureBackdrop("", wallpapers.Defaults())
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Popover = p.Popover.Alpha(0.58)
	})

	open := true
	var dlgRect ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+2 {
				col := ui.RGB(20, 80, 240)
				if i%2 == 1 {
					col = ui.RGB(240, 140, 20)
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 2, H: r.H}, col, 0)
			}
		}).Children(func() {
			ui.DialogBase(c.Key("pure-glass-dlg"), &open, func(backdrop, panel ui.Element) {
				dlgRect = DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 140, 12).Children(func() {
					ui.Box(c).Size(100, 50)
				}).Bounds()
			})
		})
	}, 260, 180)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	// Mid-enter (progress ~0.5): mask dims without fullscreen blur and panel
	// stays opaque so 40px backdrop blur sampling remains crisp.
	tt.SetTime(start.Add(OverlayDuration / 2))
	raw := testChannelSpan(tt, 10, 10, 40)
	if raw < 120 {
		t.Fatalf("expected pure dim scrim without fullscreen blur mid-enter, got stripe span=%d", raw)
	}
	insideSpan := testChannelSpan(tt, int(dlgRect.Y+dlgRect.H/2), int(dlgRect.X+20), int(dlgRect.X+dlgRect.W-20))
	if insideSpan > 30 {
		t.Fatalf("expected 40px dialog glass blur mid-enter under pure design.GlassEnabled(), got span=%d", insideSpan)
	}
}

func TestMenuSolidityCurveDialogPanelLayerAndFloatingCardLv2(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	for _, g := range []float64{20, 60, 80, 95} {
		p := wallpapers.Defaults()
		p.Glass = g
		for _, dark := range []bool{false, true} {
			if got, want := MaterialAlpha(MaterialMenu, p, dark), MenuSolidity(p, dark); got != want {
				t.Fatalf("MaterialAlpha(MaterialMenu, g=%.0f, dark=%v) = %v, want %v", g, dark, got, want)
			}
		}
	}

	url := pngDataURL(t, color.RGBA{R: 200, G: 200, B: 200, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Glass = 60
	prefs.Effect = "none"
	ConfigureBackdrop(url, prefs)

	open := true
	var dlgRect ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			WallpaperBackgroundPainter(url, prefs, true)(p, r)
		}).Children(func() {
			ui.DialogBase(c.Key("layer2-dlg"), &open, func(backdrop, panel ui.Element) {
				dlgRect = DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 120, 12).Children(func() {
					ui.Box(c).Size(80, 40)
				}).Bounds()
			})
		})
	}, 240, 160)
	start := time.Unix(2000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	// Under 60% glass, MaterialPanel alpha (0.60) is distinct from MenuSolidity (0.525).
	gotPixel := tt.Image().RGBAAt(int(dlgRect.X+dlgRect.W/2), int(dlgRect.Y+dlgRect.H/2))
	if gotPixel.R == 0 {
		t.Fatal("expected rendered dialog pixel")
	}

	var floatBox ui.Rect
	ttFloat := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			floatBox = FloatingCardSurface(ui.Box(c).Absolute().Left(20).Top(20).Width(100).Height(60), MenuBaseFill(theme.Colors{}, false), ui.RGBA(0, 0, 0, 0.1), 28).Bounds()
		})
	}, 160, 120)
	ttFloat.Frame()
	if floatBox.W != 100 || floatBox.H != 60 {
		t.Fatalf("unexpected FloatingCardSurface bounds: %+v", floatBox)
	}
}

func TestLightWallpaperClarityAndDynamicRange(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	if got := MaskAlpha(false); got != 0 {
		t.Fatalf("MaskAlpha(false) = %v, want 0 (no black fog veil in light mode)", got)
	}
	if got := MaskAlpha(true); got != 0.5 {
		t.Fatalf("MaskAlpha(true) = %v, want 0.5", got)
	}
	prefs := wallpapers.Defaults()
	prefs.Glass = 80
	prefs.Effect = "none"
	prefs.HideSidebarMask = false

	renderSample := func(wall color.RGBA, mat Material, base ui.Color, paintGlass bool) color.RGBA {
		t.Helper()
		url := pngDataURL(t, wall, 8, 8)
		tt := ui.NewTester(func(c *ui.Context) {
			paint := WallpaperBackgroundPainter(url, prefs, true)
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) }).Children(func() {
				if paintGlass {
					GlassSurface(ui.Box(c).Fill(), base, mat)
				}
			})
		}, 40, 24)
		tt.Frame()
		return tt.Image().RGBAAt(20, 12)
	}

	rawWhite := renderSample(color.RGBA{R: 255, G: 255, B: 255, A: 255}, MaterialCanvas, ui.RGB(255, 255, 255), false)
	rawBlack := renderSample(color.RGBA{R: 0, G: 0, B: 0, A: 255}, MaterialCanvas, ui.RGB(255, 255, 255), false)
	if rawWhite.R < 253 || rawBlack.R > 2 {
		t.Fatalf("light raw wallpaper unexpectedly dimmed/lifted: white=%v black=%v", rawWhite, rawBlack)
	}

	canvasWhite := renderSample(color.RGBA{R: 255, G: 255, B: 255, A: 255}, MaterialCanvas, ui.RGB(255, 255, 255), true)
	canvasBlack := renderSample(color.RGBA{R: 0, G: 0, B: 0, A: 255}, MaterialCanvas, ui.RGB(255, 255, 255), true)
	dynamicRange := float64(int(canvasWhite.R)-int(canvasBlack.R)) / 255.0
	if dynamicRange < 0.70 {
		t.Fatalf("light canvas dynamic range retention %.2f < 0.70 (white=%v black=%v)", dynamicRange, canvasWhite, canvasBlack)
	}
	if canvasBlack.R > 68 {
		t.Fatalf("light canvas over pure black washed out to gray %d (want <= 68)", canvasBlack.R)
	}

	sidebarBlack := renderSample(color.RGBA{R: 0, G: 0, B: 0, A: 255}, MaterialSidebar, ui.RGB(249, 250, 251), true)
	if sidebarBlack.R > 150 {
		t.Fatalf("light single-layer sidebar over black washed out to %d (want <= 150)", sidebarBlack.R)
	}

	// Verify light-mode MaterialAlphaFor caps popup/sidebar boost at +0.10
	// (hard ceiling <= 0.56) and never washes out to 0.74+ opaque white fog.
	secondary := Color(theme.SecondaryInk(false))
	for _, mat := range []Material{MaterialSidebar, MaterialMenu} {
		a := MaterialAlphaFor(ui.RGB(0, 0, 0), ui.RGB(255, 255, 255), mat, prefs, false, secondary)
		if a > 0.56+1e-4 {
			t.Fatalf("light MaterialAlphaFor(mat=%d) over pure black = %.2f > 0.56 (excessive white fog)", mat, a)
		}
	}
	panelA := MaterialAlphaFor(ui.RGB(0, 0, 0), ui.RGB(255, 255, 255), MaterialPanel, prefs, false, secondary)
	if panelA > MaterialAlpha(MaterialPanel, prefs, false)+1e-4 {
		t.Fatalf("light MaterialAlphaFor(MaterialPanel) boosted past token %.2f to %.2f", MaterialAlpha(MaterialPanel, prefs, false), panelA)
	}
}

func TestSettingsPanelGlassUsesGPUInGradientModeWithoutCPUCache(t *testing.T) {
	t.Cleanup(func() {
		ConfigureBackdrop("", wallpapers.Defaults())
		gradientPanelFrames.reset()
		clearGradientBackdrop()
	})
	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	prefs.Glass = 80
	// Even if a legacy gradientFrame were published, SettingsPanelGlass must
	// stay on the GPU OpEffect blur path and never populate gradientPanelFrames.
	gradientPanelFrames.reset()
	cfg := gradientConfiguration(prefs, 200, 140, false, true)
	fakeFrame := &gradientFrame{pixels: renderGradient(t.Context(), cfg, 0), config: cfg}
	publishGradientBackdrop(fakeFrame, ui.Rect{W: 200, H: 140})
	backgroundState.active = true
	backgroundState.prefs = prefs

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+4 {
				col := ui.RGB(20, 120, 240)
				if i%2 == 1 {
					col = ui.RGB(240, 120, 20)
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 4, H: r.H}, col, 0)
			}
		}).Children(func() {
			ui.Box(c).Absolute().Left(30).Top(20).Width(120).Height(80).
				Draw(SettingsPanelGlass(design.TW.White, design.TW.White, ui.RGBA(18, 22, 30, 0.12), 16))
		})
	}, 200, 140)
	tt.Frame()
	if span := testChannelSpan(tt, 60, 50, 130); span > 35 {
		t.Fatalf("SettingsPanelGlass did not run GPU backdrop blur: span=%d", span)
	}
	gradientPanelFrames.mu.Lock()
	usedCPU := gradientPanelFrames.satBlur != nil || gradientPanelFrames.rgba != nil
	for _, e := range gradientPanelFrames.entries {
		if e.bitmap != nil {
			usedCPU = true
		}
	}
	gradientPanelFrames.mu.Unlock()
	if usedCPU {
		t.Fatal("SettingsPanelGlass unexpectedly populated CPU gradientPanelFrames cache")
	}
}
