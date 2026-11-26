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
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
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

	// Light mode with no wallpaper and no gradient must keep design.GlassEnabled()
	// false even when GlassTheme is on, avoiding glass rims/shadows on flat white.
	cfg := minimalModeStore(t)
	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}
	applyGlassTheme(true)
	ui.NewTester(func(c *ui.Context) { s.beginAppearanceFrame(c) }, 320, 200)
	if design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = true on flat light window without backdrop")
	}

	// Enabling a gradient backdrop activates design.SetGlass, while minimal mode
	// and SkinCodex disable it cleanly and identical frames do not churn Revision().
	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	s.wallpaperControls = nil
	ui.NewTester(func(c *ui.Context) { s.beginAppearanceFrame(c) }, 320, 200)
	if !design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = false with gradient backdrop and glass theme on")
	}

	applyMinimalMode(true)
	if design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = true while minimal mode is on")
	}

	applyMinimalMode(false)
	ui.NewTester(func(c *ui.Context) { s.beginAppearanceFrame(c) }, 320, 200)
	if !design.GlassEnabled() {
		t.Fatal("design.GlassEnabled() = false after leaving minimal mode with gradient backdrop")
	}

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

func TestDarkAndLightGlassPolishAndUnifiedSettingsModal(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	// Mid-saturation solid backdrop lets us verify both glass translucency/tints
	// and that the settings modal rail has no opaque double-layer seam.
	wallURL := testSolidWallpaperURL(t, color.RGBA{R: 60, G: 110, B: 190, A: 255}, 64, 64)
	cfg := minimalModeStore(t)
	if err := cfg.Update(func(c *config.Config) {
		c.Theme = theme.IDDeepseekDark
		c.WallpaperPath = wallURL
		on := true
		c.GlassTheme = &on
	}); err != nil {
		t.Fatal(err)
	}
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Glass = 60
	prefs.HideSidebarMask = false
	prefs.Blur = 40
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}

	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekDark}
	store := session.New()
	store.Select("fixture")
	sizes := frame.DefaultLayout()
	sizes.Bottom = 0
	draft := ""
	showSettings := true
	var gotTokens design.Tokens

	tt := ui.NewTester(func(c *ui.Context) {
		s.beginAppearanceFrame(c)
		defer s.endAppearanceFrame()
		gotTokens = *design.For(c)

		wp := s.currentWallpaperConfigPath()
		wallpaperDraw := s.wallpaperPainter(wp, true, c.Preferences().ReduceMotion)
		m := frame.Model{
			Store:              store,
			Sizes:              &sizes,
			Draft:              &draft,
			ThemeID:            s.themeID,
			SystemDark:         s.themeID == theme.IDDeepseekDark,
			ExternalNavigation: true,
		}
		ui.Column(c).Fill().Background(chrome.Color(theme.Resolve(s.themeID).Active(m.SystemDark).BG)).Draw(wallpaperDraw).Children(func() {
			frame.NavigationShell(c, m, func() {
				frame.View(c, m)
			})
			if showSettings {
				settings.ModalView(c, settings.Model{
					ActiveSection: settings.SectionGeneral,
					ThemeID:       s.themeID,
					SystemDark:    m.SystemDark,
					SectionContent: func(c *ui.Context) {
						ui.Box(c).Size(8, 8)
					},
					OnClose: func() {},
				})
			}
		})
	}, 1100, 680)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1002, 0))

	renderSettled := func(dark bool) *image.RGBA {
		t.Helper()
		tt.SetDark(dark)
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
		return tt.Image()
	}

	findDialogRect := func() ui.Rect {
		t.Helper()
		for _, b := range tt.Interactives() {
			if b.Role == ui.RoleDialog {
				return b.Rect
			}
		}
		t.Fatal("settings dialog not found in interactives")
		return ui.Rect{}
	}

	for _, tc := range []struct {
		name    string
		themeID string
		dark    bool
	}{
		{"dark", theme.IDDeepseekDark, true},
		{"light", theme.IDDeepseekLight, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.themeID = tc.themeID
			applyGlassTheme(true)
			img := renderSettled(tc.dark)

			if !gotTokens.Glass {
				t.Fatalf("expected Glass=true tokens in %s mode", tc.name)
			}
			if tc.dark {
				if gotTokens.Code != ui.RGB(27, 27, 28) {
					t.Fatalf("dark Code = %+v, want #1b1b1c", gotTokens.Code)
				}
				if gotTokens.Border != ui.RGBA(255, 255, 255, 0.14) {
					t.Fatalf("dark Border = %+v, want rgba(255,255,255,0.14)", gotTokens.Border)
				}
				if gotTokens.SidebarBorder != ui.RGBA(255, 255, 255, 0.08) {
					t.Fatalf("dark SidebarBorder = %+v, want rgba(255,255,255,0.08)", gotTokens.SidebarBorder)
				}
				if gotTokens.Input != ui.RGBA(255, 255, 255, 0.16) {
					t.Fatalf("dark Input = %+v, want rgba(255,255,255,0.16)", gotTokens.Input)
				}
			} else {
				if gotTokens.Code != ui.RGB(249, 250, 251) {
					t.Fatalf("light Code = %+v, want #f9fafb", gotTokens.Code)
				}
				if gotTokens.Border != ui.RGBA(0, 0, 0, 0.10) {
					t.Fatalf("light Border = %+v, want rgba(0,0,0,0.10)", gotTokens.Border)
				}
				if gotTokens.SidebarBorder != ui.RGBA(0, 0, 0, 0.07) {
					t.Fatalf("light SidebarBorder = %+v, want rgba(0,0,0,0.07)", gotTokens.SidebarBorder)
				}
				if gotTokens.Input != ui.RGBA(0, 0, 0, 0.12) {
					t.Fatalf("light Input = %+v, want rgba(0,0,0,0.12)", gotTokens.Input)
				}
			}

			dlg := findDialogRect()
			// Left rail background (empty bottom area of the 208-DIP rail) vs right
			// content pane background: both must sit on the unified glass panel with
			// only a subtle wash difference, no heavy double-layered seam.
			railPx := img.RGBAAt(int(dlg.X)+24, int(dlg.Y+dlg.H)-24)
			panePx := img.RGBAAt(int(dlg.X)+260, int(dlg.Y+dlg.H)-24)
			channelDiff := max(
				int(math.Abs(float64(railPx.R)-float64(panePx.R))),
				max(
					int(math.Abs(float64(railPx.G)-float64(panePx.G))),
					int(math.Abs(float64(railPx.B)-float64(panePx.B))),
				),
			)
			if channelDiff > 22 {
				t.Fatalf("%s settings modal rail vs pane seam too strong (rail=%+v pane=%+v maxDiff=%d)", tc.name, railPx, panePx, channelDiff)
			}

			// Glass border & 1-DIP inner rim highlight on top edge of settings modal.
			midX := int(dlg.X + dlg.W/2)
			outerBorder := img.RGBAAt(midX, int(dlg.Y))
			innerRim := img.RGBAAt(midX, int(dlg.Y)+1)
			interior := img.RGBAAt(midX, int(dlg.Y)+16)
			if tc.dark {
				if innerRim.R <= interior.R || innerRim.B <= interior.B {
					t.Fatalf("dark modal inner rim highlight not brighter than interior: rim=%+v interior=%+v", innerRim, interior)
				}
				if outerBorder.R <= interior.R {
					t.Fatalf("dark modal outer border not visible over interior: border=%+v interior=%+v", outerBorder, interior)
				}
			} else {
				if innerRim.R <= outerBorder.R {
					t.Fatalf("light modal inner rim highlight not brighter than outer border: rim=%+v border=%+v", innerRim, outerBorder)
				}
			}

			// Turning GlassTheme OFF restores 100% original opaque coss tokens & rail/pane contrast.
			applyGlassTheme(false)
			tt.Frame()
			offImg := tt.Image()
			if gotTokens.Glass {
				t.Fatalf("expected Glass=false after applyGlassTheme(false) in %s mode", tc.name)
			}
			scheme := design.Light
			if tc.dark {
				scheme = design.Dark
			}
			base := design.Base(scheme)
			if gotTokens.Border != base.Border || gotTokens.Popover != base.Popover || gotTokens.Sidebar != base.Sidebar {
				t.Fatalf("%s glass-off tokens diverged from base: border=%+v popover=%+v sidebar=%+v", tc.name, gotTokens.Border, gotTokens.Popover, gotTokens.Sidebar)
			}
			offRail := offImg.RGBAAt(int(dlg.X)+24, int(dlg.Y+dlg.H)-24)
			offPane := offImg.RGBAAt(int(dlg.X)+260, int(dlg.Y+dlg.H)-24)
			if offRail.R != base.Sidebar.R || offRail.G != base.Sidebar.G || offRail.B != base.Sidebar.B {
				t.Fatalf("%s glass-off settings rail = %+v, want opaque base.Sidebar %+v", tc.name, offRail, base.Sidebar)
			}
			if offPane.R != base.Popover.R || offPane.G != base.Popover.G || offPane.B != base.Popover.B {
				t.Fatalf("%s glass-off settings pane = %+v, want opaque base.Popover %+v", tc.name, offPane, base.Popover)
			}
		})
	}
}

func TestLightFlatNoBackdropDisablesGlassAndAcrylicCurvesMatchReference(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	// 1. Light mode without wallpaper or gradient must yield t.Glass == false.
	cfg := minimalModeStore(t)
	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}
	var gotLight design.Tokens
	ui.NewTester(func(c *ui.Context) {
		s.beginAppearanceFrame(c)
		gotLight = *design.For(c)
	}, 320, 200)
	if gotLight.Glass {
		t.Fatal("expected t.Glass == false on light mode without wallpaper/gradient")
	}

	// 2. Verify Windows Acrylic curve (g=80% -> 0.30 canvas, ShowThroughAlpha sidebar,
	// 0.80 card, 0.64 dark popover) and light menu opacity (g=80% -> 0.70).
	prefs := wallpapers.Defaults()
	prefs.Glass = 80
	if got := acrylicCanvasAlpha(prefs); math.Abs(float64(got)-0.30) > 0.001 {
		t.Fatalf("acrylicCanvasAlpha(80) = %.4f, want 0.30", got)
	}
	if got := darkGlassMenuAlpha(prefs); math.Abs(float64(got)-0.64) > 0.001 {
		t.Fatalf("darkGlassMenuAlpha(80) = %.4f, want 0.64", got)
	}
	if got := lightGlassMenuAlpha(prefs); math.Abs(float64(got)-0.70) > 0.001 {
		t.Fatalf("lightGlassMenuAlpha(80) = %.4f, want 0.70", got)
	}

	filter := glassSurfaceFilter(designGlassSignature{
		enabled: true,
		dark:    true,
		glass:   80,
		acrylic: true,
	})
	pal := design.Base(design.Dark)
	filter(design.Dark, &pal)
	if pal.Background != chrome.MaterialFill(ui.RGB(21, 21, 23), 0.30) {
		t.Fatalf("Acrylic Background = %+v, want #151517 @ 0.30", pal.Background)
	}
	if pal.Sidebar != chrome.MaterialFill(ui.RGB(27, 27, 28), chrome.ShowThroughAlpha(prefs, true)) {
		t.Fatalf("Acrylic Sidebar = %+v, want #1b1b1c @ ShowThroughAlpha", pal.Sidebar)
	}
	if pal.Card != chrome.MaterialFill(ui.RGB(35, 35, 36), 0.80) {
		t.Fatalf("Acrylic Card = %+v, want #232324 @ 0.80", pal.Card)
	}
	if pal.Popover != chrome.MaterialFill(ui.RGB(48, 49, 54), 0.64) {
		t.Fatalf("Acrylic Popover = %+v, want rgb(48,49,54) @ 0.64", pal.Popover)
	}
}

func TestRightPaneDecoupledFromHideSidebarMask(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	cfg := minimalModeStore(t)
	stripedURL := testStripedWallpaperURL(t, 300, 200)
	if err := cfg.Update(func(c *config.Config) {
		c.Theme = theme.IDDeepseekDark
		c.WallpaperPath = stripedURL
		on := true
		c.GlassTheme = &on
	}); err != nil {
		t.Fatal(err)
	}
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Glass = 80
	prefs.HideSidebarMask = true
	prefs.Transparent = false
	prefs.Blur = 0
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}

	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekDark}
	store := session.New()
	store.Select("fixture")
	sizes := frame.DefaultLayout()
	sizes.Bottom = 0
	draft := ""

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
			SystemDark:         true,
			ExternalNavigation: true,
			RightPaneOpen:      true,
		}
		ui.Column(c).Fill().Background(chrome.Color(theme.Resolve(s.themeID).Active(true).BG)).Draw(wallpaperDraw).Children(func() {
			frame.NavigationShell(c, m, func() {
				frame.View(c, m)
			})
		})
	}, 1100, 680)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if _, ok := chrome.SampleBackdrop(ui.Rect{X: 0, Y: 0, W: 40, H: 40}); ok {
			tt.Frame()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	img := tt.Image()

	// Measure horizontal color variation across the right pane (x=920..1040, y=500):
	// frosted glass blur must keep stripe variation low even though HideSidebarMask=true.
	loR, hiR := 255, 0
	loB, hiB := 255, 0
	for x := 920; x < 1040; x++ {
		px := img.RGBAAt(x, 500)
		loR, hiR = min(loR, int(px.R)), max(hiR, int(px.R))
		loB, hiB = min(loB, int(px.B)), max(hiB, int(px.B))
	}
	rightSpan := max(hiR-loR, hiB-loB)

	// Left sidebar (x=40..160, y=500) with HideSidebarMask=true has no blur,
	// so raw green/blue stripes retain high contrast.
	lLoG, lHiG := 255, 0
	lLoB, lHiB := 255, 0
	for x := 40; x < 160; x++ {
		px := img.RGBAAt(x, 500)
		lLoG, lHiG = min(lLoG, int(px.G)), max(lHiG, int(px.G))
		lLoB, lHiB = min(lLoB, int(px.B)), max(lHiB, int(px.B))
	}
	leftSpan := max(lHiG-lLoG, lHiB-lLoB)

	if rightSpan >= leftSpan/2 || rightSpan > 25 {
		t.Fatalf("right pane failed frosted blur decoupling under HideSidebarMask=true: rightSpan=%d leftSpan=%d", rightSpan, leftSpan)
	}
}

func TestToolCodeBlockNoDoubleBlendOnStickyBanner(t *testing.T) {
	translucentCode := ui.RGBA(27, 27, 28, 0.85)
	st := markdown.Style{
		FontSize: 14,
		FG:       ui.RGB(235, 235, 240),
		Muted:    ui.RGB(150, 150, 160),
		CodeBG:   translucentCode,
		CodeSurface: func(e ui.Element, radius float32) ui.Element {
			return e.Background(translucentCode).Border(1, ui.RGBA(255, 255, 255, 0.14)).Radius(radius)
		},
		Dark: true,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(40, 120, 220)).Padding(20).Children(func() {
			markdown.ToolCodeBlock(c, "go", "x := 1\ny := 2\nz := 3", st, 0)
		})
	}, 420, 260)
	tt.Frame()
	img := tt.Image()

	// Sample inside the top 42px banner (away from text/buttons) and inside the
	// lower card body (away from text): both must match exactly (single-layer fill).
	topPx := img.RGBAAt(220, 32)
	bodyPx := img.RGBAAt(360, 100)
	if topPx != bodyPx {
		t.Fatalf("ToolCodeBlock sticky banner double-blended top 42px: banner=%+v body=%+v", topPx, bodyPx)
	}
}

func TestSettingsDunePresetSelectable(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)

	var saved wallpapers.Preferences
	g := settings.NewWallpaperEditor(wallpapers.Defaults(), "", func(p wallpapers.Preferences) error {
		saved = p
		return nil
	}, nil, nil)
	path := ""
	glassOn := true
	a := &settings.AppearanceAccess{
		Glass:             &settings.GlassThemeAccess{Enabled: &glassOn},
		Background:        g,
		WallpaperPath:     &path,
		OnBrowseWallpaper: func() {},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		settings.ModalView(c, settings.Model{
			Desktop:          &settings.DesktopAccess{},
			ActiveSection:    settings.SectionAppearance,
			AppearanceActive: true,
			Appearance:       a,
			ThemeID:          theme.IDDeepseekLight,
			OnClose:          func() {},
		})
	}, 960, 1400)
	tt.Frame()

	if err := tt.Click("Background effect settings"); err != nil {
		t.Fatalf("click Background effect settings: %v", err)
	}
	tt.Frame()
	if !tt.HasText("Dune") {
		t.Fatal("expected Dune preset option in background effect settings")
	}
	if err := tt.Click("Dune"); err != nil {
		t.Fatalf("click Dune preset: %v", err)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("click Save: %v", err)
	}
	if saved.Preset != "dune" {
		t.Fatalf("saved.Preset = %q, want dune", saved.Preset)
	}
	if saved.Colors != wallpapers.PresetColors("dune") {
		t.Fatalf("saved.Colors = %+v, want PresetColors(\"dune\")", saved.Colors)
	}
}

func TestSettingsBodyDialogFrameGlassBlurAndRim(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	// Enable glass tokens via an active wallpaper backdrop and verify that
	// bodyDialogFrame (via settings.ConfirmDialog) blurs underlying sharp stripes
	// instead of letting underlying UI details bleed through unblurred.
	stripedURL := testStripedWallpaperURL(t, 300, 200)
	cfg := minimalModeStore(t)
	if err := cfg.Update(func(c *config.Config) {
		c.Theme = theme.IDDeepseekLight
		c.WallpaperPath = stripedURL
		on := true
		c.GlassTheme = &on
	}); err != nil {
		t.Fatal(err)
	}
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Glass = 80
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}

	tt := ui.NewTester(func(c *ui.Context) {
		s.beginAppearanceFrame(c)
		defer s.endAppearanceFrame()

		// Paint crisp alternating 4px high-contrast stripes underneath the dialog.
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			c1, c2 := ui.RGB(20, 110, 240), ui.RGB(245, 130, 30)
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+4 {
				col := c1
				if i%2 == 1 {
					col = c2
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 4, H: r.H}, col, 0)
			}
		}).Children(func() {
			settings.ConfirmDialog(c, settings.ConfirmOptions{
				Title:        "Glass dialog",
				ConfirmLabel: "OK",
				CancelLabel:  "Cancel",
				Width:        360,
			})
		})
	}, 600, 420)
	tt.Frame()
	if !design.GlassEnabled() {
		t.Fatal("expected design.GlassEnabled() == true")
	}
	img := tt.Image()

	var dlg ui.Rect
	for _, b := range tt.Interactives() {
		if b.Role == ui.RoleDialog {
			dlg = b.Rect
			break
		}
	}
	if dlg.W <= 0 || dlg.H <= 0 {
		t.Fatal("ConfirmDialog not found in interactives")
	}

	// Inside the blurred dialog body, stripe contrast must be dramatically softened
	// compared to the raw unblurred stripes outside the backdrop scrim.
	midY := int(dlg.Y + dlg.H/2)
	loR, hiR := 255, 0
	loB, hiB := 255, 0
	for x := int(dlg.X) + 32; x < int(dlg.X+dlg.W)-32; x++ {
		px := img.RGBAAt(x, midY)
		loR, hiR = min(loR, int(px.R)), max(hiR, int(px.R))
		loB, hiB = min(loB, int(px.B)), max(hiB, int(px.B))
	}
	insideSpan := max(hiR-loR, hiB-loB)
	if insideSpan > 20 {
		t.Fatalf("bodyDialogFrame did not blur underlying stripes enough: insideSpan=%d", insideSpan)
	}

	// Turning GlassTheme off restores opaque Popover with zero stripe bleed.
	applyGlassTheme(false)
	tt.Frame()
	offImg := tt.Image()
	offLoR, offHiR := 255, 0
	for x := int(dlg.X) + 32; x < int(dlg.X+dlg.W)-32; x++ {
		px := offImg.RGBAAt(x, midY)
		offLoR, offHiR = min(offLoR, int(px.R)), max(offHiR, int(px.R))
	}
	if offHiR-offLoR != 0 {
		t.Fatalf("expected opaque bodyDialogFrame when glass is off, got R span=%d", offHiR-offLoR)
	}
}

func TestWallpaperPrecedenceAndTransparentThemeContract(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	// 1. Gradient without wallpaper: Transparent=true stays inert (does not drop
	// backdrop glass opacity as transparent wallpaper theme).
	cfg := minimalModeStore(t)
	if err := cfg.Update(func(c *config.Config) {
		c.Theme = theme.IDDeepseekLight
		c.WallpaperPath = ""
		on := true
		c.GlassTheme = &on
	}); err != nil {
		t.Fatal(err)
	}
	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Transparent = true
	prefs.Glass = 80
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}
	s.syncAppearanceBackdrop(theme.IDDeepseekLight, false, false)
	if !chrome.BackdropActive() || chrome.BackdropHasWallpaper() {
		t.Fatalf("gradient-only state wrong: active=%v hasWallpaper=%v", chrome.BackdropActive(), chrome.BackdropHasWallpaper())
	}
	// Transparent theme over gradient without wallpaper must keep non-zero canvas fill.
	if bg := design.Base(design.Light).Background; bg.A == 0 {
		t.Fatalf("expected Transparent=true to be inert without wallpaper, got Background.A=0")
	}

	// 2. Wallpaper image takes precedence over gradient, applies dim mask
	// (0.24 light / 0.50 dark), and Transparent=true drops the mask only when
	// a wallpaper is active.
	if math.Abs(float64(chrome.MaskAlpha(false)-0.24)) > 1e-6 || math.Abs(float64(chrome.MaskAlpha(true)-0.50)) > 1e-6 {
		t.Fatalf("unexpected MaskAlpha values: light=%v dark=%v", chrome.MaskAlpha(false), chrome.MaskAlpha(true))
	}
	stripedURL := testStripedWallpaperURL(t, 120, 80)
	opaquePrefs := prefs
	opaquePrefs.Transparent = false
	paintOne := func(p wallpapers.Preferences) color.RGBA {
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.LightTheme())
			ui.Box(c).Fill().Draw(chrome.WallpaperBackgroundPainter(stripedURL, p, false))
		}, 120, 80)
		for i := 0; i < 60; i++ {
			tt.Frame()
			px := tt.Image().RGBAAt(10, 10)
			if px != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) && px != (color.RGBA{A: 255}) {
				return px
			}
			time.Sleep(5 * time.Millisecond)
		}
		return tt.Image().RGBAAt(10, 10)
	}
	dimPx := paintOne(opaquePrefs)
	transPx := paintOne(prefs)
	if dimPx == (color.RGBA{R: 255, G: 255, B: 255, A: 255}) || int(transPx.R)+int(transPx.G)+int(transPx.B) <= int(dimPx.R)+int(dimPx.G)+int(dimPx.B) {
		t.Fatalf("expected Transparent=true to drop 0.24 dim mask over wallpaper: dim=%+v trans=%+v", dimPx, transPx)
	}

	// 3. Turning off the optional glass theme switch disables backdrop & glass sampling.
	applyGlassTheme(false)
	if !renderpolicy.NoBackdrop() || design.GlassEnabled() {
		t.Fatalf("expected glass theme off to enforce NoBackdrop and disable design glass: noBackdrop=%v glass=%v", renderpolicy.NoBackdrop(), design.GlassEnabled())
	}
}

func TestLightGlassScrollbarTranslucentContract(t *testing.T) {
	resetAppearanceGlobalsForTest(t)
	applyMinimalMode(false)
	applyGlassTheme(true)

	cfg := minimalModeStore(t)
	if err := cfg.Update(func(c *config.Config) {
		c.Theme = theme.IDDeepseekLight
		c.WallpaperPath = ""
		on := true
		c.GlassTheme = &on
	}); err != nil {
		t.Fatal(err)
	}
	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}

	var gotRest, gotHover ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		s.beginAppearanceFrame(c)
		defer s.endAppearanceFrame()
		gotRest = c.Theme().Scrollbar
		gotHover = c.Theme().ScrollbarHover
	}, 200, 120)
	tt.Frame()

	wantRest := ui.RGBA(15, 17, 21, 0.18)
	wantHover := ui.RGBA(15, 17, 21, 0.30)
	if gotRest != wantRest || gotHover != wantHover {
		t.Fatalf("light glass scrollbars = (%+v, %+v), want (%+v, %+v)", gotRest, gotHover, wantRest, wantHover)
	}

	// Dark glass keeps dark scrollbar colors instead of light neutral-bluish-1000.
	s.themeID = theme.IDDeepseekDark
	_ = cfg.Update(func(c *config.Config) { c.Theme = theme.IDDeepseekDark })
	tt.Frame()
	darkRest, darkHover := theme.ScrollbarColors(true)
	if gotRest != chrome.Color(darkRest) || gotHover != chrome.Color(darkHover) {
		t.Fatalf("dark glass scrollbars = (%+v, %+v), want (%+v, %+v)", gotRest, gotHover, chrome.Color(darkRest), chrome.Color(darkHover))
	}

	// Glass theme off under light scheme restores solid ScrollbarColors(false).
	s.themeID = theme.IDDeepseekLight
	_ = cfg.Update(func(c *config.Config) {
		c.Theme = theme.IDDeepseekLight
		off := false
		c.GlassTheme = &off
	})
	applyGlassTheme(false)
	tt.Frame()
	lightRest, lightHover := theme.ScrollbarColors(false)
	if gotRest != chrome.Color(lightRest) || gotHover != chrome.Color(lightHover) {
		t.Fatalf("non-glass light scrollbars = (%+v, %+v), want (%+v, %+v)", gotRest, gotHover, chrome.Color(lightRest), chrome.Color(lightHover))
	}
}
