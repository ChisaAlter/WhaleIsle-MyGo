package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

func testStripedWallpaperURL(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var c color.RGBA
			if x < w/3 {
				// Left band (under sidebar): vivid green vs vivid blue stripes.
				if (x/8+y/8)%2 == 0 {
					c = color.RGBA{R: 20, G: 220, B: 80, A: 255}
				} else {
					c = color.RGBA{R: 20, G: 80, B: 235, A: 255}
				}
			} else if x < 2*w/3 {
				// Center band (under main chat / settings modal): vivid red vs cyan stripes.
				if (x/8+y/8)%2 == 0 {
					c = color.RGBA{R: 235, G: 55, B: 45, A: 255}
				} else {
					c = color.RGBA{R: 35, G: 215, B: 230, A: 255}
				}
			} else {
				// Right band (under right pane): vivid magenta vs amber stripes.
				if (x/8+y/8)%2 == 0 {
					c = color.RGBA{R: 215, G: 45, B: 210, A: 255}
				} else {
					c = color.RGBA{R: 245, G: 185, B: 30, A: 255}
				}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func testSolidWallpaperURL(t *testing.T, c color.RGBA, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func resetAppearanceGlobalsForTest(t *testing.T) {
	t.Helper()
	prevPrefs := theme.CurrentPreferences()
	t.Cleanup(func() {
		theme.SetPreferences(prevPrefs)
		applyMinimalMode(false)
		applyGlassTheme(false)
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		chrome.SetBackdropDark(false)
		design.SetGlass(false, nil)
	})
}

func TestApplyGlassThemeAndMinimalModeSyncDesignGlass(t *testing.T) {
	resetAppearanceGlobalsForTest(t)

	applyMinimalMode(false)
	applyGlassTheme(false)
	if design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = true with glass theme off")
	}

	applyGlassTheme(true)
	if !design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = false after applyGlassTheme(true)")
	}

	applyMinimalMode(true)
	if design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = true while minimal mode is on")
	}

	applyMinimalMode(false)
	if !design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = false after leaving minimal mode with glass theme on")
	}

	// Codex flat skin disables design.SetGlass, and identical frames do not
	// churn design.Revision().
	cfg := minimalModeStore(t)
	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}
	p := theme.Defaults()
	p.Light = theme.SkinCodex
	theme.SetPreferences(p)
	ui.NewTester(func(c *ui.Context) { s.beginAppearanceFrame(c) }, 320, 200)
	if design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = true under SkinCodex")
	}

	p.Light = theme.SkinDefault
	theme.SetPreferences(p)
	tt := ui.NewTester(func(c *ui.Context) { s.beginAppearanceFrame(c) }, 320, 200)
	if !design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = false after restoring default skin")
	}
	revAfterFirst := design.Revision()
	for range 5 {
		tt.Frame()
	}
	if got := design.Revision(); got != revAfterFirst {
		t.Fatalf("design.Revision() churned across unchanged frames: %d -> %d", revAfterFirst, got)
	}
}

func TestBackgroundDarkUpdatesWithoutWallpaperImage(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	cfg := minimalModeStore(t)
	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}

	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekDark}
	ui.NewTester(func(c *ui.Context) { s.beginAppearanceFrame(c) }, 320, 200)
	if !chrome.BackdropDark() {
		t.Fatal("chrome.BackdropDark() = false in dark mode with gradient and no image")
	}

	s.themeID = theme.IDDeepseekLight
	ui.NewTester(func(c *ui.Context) { s.beginAppearanceFrame(c) }, 320, 200)
	if chrome.BackdropDark() {
		t.Fatal("chrome.BackdropDark() = true in light mode")
	}
}

func TestWallpaperAndGlassThroughNavigationShellSidebarAndSettings(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	cfg := minimalModeStore(t)
	// Use a vivid blue-tinted solid wallpaper for translucency / opaque-fallback
	// verification, then a striped wallpaper for blur & SettingsPanelGlass verification.
	solidURL := testSolidWallpaperURL(t, color.RGBA{R: 30, G: 110, B: 240, A: 255}, 64, 64)
	if err := cfg.Update(func(c *config.Config) {
		c.Theme = theme.IDDeepseekLight
		c.WallpaperPath = solidURL
		on := true
		c.GlassTheme = &on
	}); err != nil {
		t.Fatal(err)
	}
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Glass = 60
	prefs.HideSidebarMask = false
	prefs.Blur = 0
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}

	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}
	store := session.New()
	store.Select("fixture")
	sizes := frame.DefaultLayout()
	sizes.Bottom = 0
	draft := ""
	showSettings := false

	renderShell := func(tt *ui.Tester, waitDecode bool) *image.RGBA {
		t.Helper()
		if waitDecode {
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				tt.Frame()
				if _, ok := chrome.SampleBackdrop(ui.Rect{X: 0, Y: 0, W: 40, H: 40}); ok {
					tt.Frame()
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if _, ok := chrome.SampleBackdrop(ui.Rect{X: 0, Y: 0, W: 40, H: 40}); !ok {
				t.Fatal("wallpaper decode did not settle within timeout")
			}
		} else {
			tt.Frame()
		}
		return tt.Image()
	}

	tt := ui.NewTester(func(c *ui.Context) {
		s.beginAppearanceFrame(c)
		defer s.endAppearanceFrame()

		wp := s.currentWallpaperConfigPath()
		wallpaperDraw := s.wallpaperPainter(wp, true, c.Preferences().ReduceMotion)
		m := frame.Model{
			Store:              store,
			Sizes:              &sizes,
			Draft:              &draft,
			ThemeID:            s.themeID,
			ExternalNavigation: true,
			RightPaneOpen:      true,
		}
		ui.Column(c).Fill().Background(chrome.Color(theme.Resolve(s.themeID).Active(false).BG)).Draw(wallpaperDraw).Children(func() {
			frame.NavigationShell(c, m, func() {
				frame.View(c, m)
			})
			if showSettings {
				settings.ModalView(c, settings.Model{
					ActiveSection: settings.SectionGeneral,
					SectionContent: func(c *ui.Context) {
						ui.Box(c).Size(8, 8)
					},
					OnClose: func() {},
				})
			}
		})
	}, 1100, 680)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	// (a) With GlassTheme ON and a vivid blue wallpaper, NavigationShell +
	// sidebar + right pane must let the blue wallpaper bleed through instead of
	// painting opaque white/gray.
	glassImg := renderShell(tt, true)
	sidebarPx := glassImg.RGBAAt(40, 500)
	centerPx := glassImg.RGBAAt(520, 120)
	rightPx := glassImg.RGBAAt(960, 200)
	if sidebarPx.B <= sidebarPx.R+25 {
		t.Fatalf("sidebar obscured blue wallpaper: pixel=%+v", sidebarPx)
	}
	if centerPx.B <= centerPx.R+40 {
		t.Fatalf("navigation shell center obscured blue wallpaper: pixel=%+v", centerPx)
	}
	if rightPx.B <= rightPx.R+25 {
		t.Fatalf("right pane obscured blue wallpaper: pixel=%+v", rightPx)
	}

	// (c) Turning GlassTheme OFF restores 100% opaque coss default surfaces.
	applyGlassTheme(false)
	offImg := renderShell(tt, false)
	wantCenter := design.Base(design.Light).Background
	wantSide := design.Base(design.Light).Sidebar
	gotOffCenter := offImg.RGBAAt(520, 120)
	gotOffSide := offImg.RGBAAt(40, 500)
	if gotOffCenter.R != wantCenter.R || gotOffCenter.G != wantCenter.G || gotOffCenter.B != wantCenter.B {
		t.Fatalf("glass off center = %+v, want opaque coss Background %+v", gotOffCenter, wantCenter)
	}
	if gotOffSide.R != wantSide.R || gotOffSide.G != wantSide.G || gotOffSide.B != wantSide.B {
		t.Fatalf("glass off sidebar = %+v, want opaque coss Sidebar %+v", gotOffSide, wantSide)
	}

	// Turning MinimalMode ON while GlassTheme is ON also restores opaque surfaces.
	applyGlassTheme(true)
	_ = renderShell(tt, true)
	applyMinimalMode(true)
	minImg := renderShell(tt, false)
	gotMinCenter := minImg.RGBAAt(520, 120)
	if gotMinCenter.R != wantCenter.R || gotMinCenter.G != wantCenter.G || gotMinCenter.B != wantCenter.B {
		t.Fatalf("minimal mode center = %+v, want opaque coss Background %+v", gotMinCenter, wantCenter)
	}

	// (b) Striped wallpaper: Blur > 0 blurs the background behind the shell,
	// and SettingsPanelGlass produces a frosted modal surface distinct from
	// both Blur == 0 and no-glass opaque Popover.
	applyMinimalMode(false)
	stripedURL := testStripedWallpaperURL(t, 300, 200)
	if err := cfg.Update(func(c *config.Config) { c.WallpaperPath = stripedURL }); err != nil {
		t.Fatal(err)
	}
	s.wallpaperControls = nil
	prefs.Blur = 0
	prefs.Glass = 25
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	blur0Img := renderShell(tt, true)

	prefs.Blur = 80
	s.wallpaperControls = nil
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	blur80Img := renderShell(tt, true)

	var blurDiffPixels int
	for y := 80; y < 320; y += 2 {
		for x := 340; x < 740; x += 2 {
			a, b := blur0Img.RGBAAt(x, y), blur80Img.RGBAAt(x, y)
			if math.Abs(float64(a.R)-float64(b.R))+math.Abs(float64(a.G)-float64(b.G))+math.Abs(float64(a.B)-float64(b.B)) > 6 {
				blurDiffPixels++
			}
		}
	}
	if blurDiffPixels < 500 {
		t.Fatalf("wallpaper Blur=80 vs Blur=0 produced insufficient pixel difference through shell (%d pixels)", blurDiffPixels)
	}

	// SettingsPanelGlass vs opaque no-glass modal.
	showSettings = true
	tt.SetTime(time.Unix(1002, 0))
	tt.SetTime(time.Unix(1004, 0))
	glassModalImg := renderShell(tt, true)

	// Measure color variation across the settings modal panel area: SettingsPanelGlass
	// lets the striped backdrop show through blurred & tinted, whereas no-glass
	// Popover paints a uniform flat #ffffff.
	applyGlassTheme(false)
	opaqueModalImg := renderShell(tt, false)
	if renderpolicy.Glass() {
		t.Fatal("expected renderpolicy.Glass() to be false")
	}

	var modalDiffPixels int
	for y := 120; y < 560; y += 4 {
		for x := 380; x < 840; x += 4 {
			a, b := glassModalImg.RGBAAt(x, y), opaqueModalImg.RGBAAt(x, y)
			if math.Abs(float64(a.R)-float64(b.R))+math.Abs(float64(a.G)-float64(b.G))+math.Abs(float64(a.B)-float64(b.B)) > 8 {
				modalDiffPixels++
			}
		}
	}
	if modalDiffPixels < 500 {
		t.Fatalf("SettingsPanelGlass vs opaque settings modal produced insufficient pixel difference (%d pixels)", modalDiffPixels)
	}
}
