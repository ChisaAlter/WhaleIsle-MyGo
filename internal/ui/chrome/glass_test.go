package chrome

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
)

// TestMaterialAlphaKeepsTokenSolidity pins the solidity each layer gets from
// the harness wallpaper tokens (ui-theme/src/wallpaper.ts, mixWallpaperSurfaces)
// and checks the curves only grow with the glass slider.
func TestMaterialAlphaKeepsTokenSolidity(t *testing.T) {
	for _, tc := range []struct {
		glass   float64
		canvas  float32
		sidebar float32
	}{
		{glass: 0, canvas: 0, sidebar: 0},
		{glass: 20, canvas: .075, sidebar: .1375},
		{glass: 40, canvas: .15, sidebar: .275},
		{glass: 60, canvas: .30, sidebar: .45},
		{glass: 80, canvas: .45, sidebar: .625},
		{glass: 100, canvas: .45, sidebar: 1},
	} {
		p := wallpapers.Defaults()
		p.Glass, p.HideSidebarMask = tc.glass, false
		for _, want := range []struct {
			m     Material
			alpha float32
		}{{MaterialCanvas, tc.canvas}, {MaterialSidebar, tc.sidebar}, {MaterialPanel, float32(tc.glass / 100)}} {
			if got := MaterialAlpha(want.m, p, false); math.Abs(float64(got-want.alpha)) > 1e-4 {
				t.Fatalf("glass %.0f material %d: alpha = %v, want %v", tc.glass, want.m, got, want.alpha)
			}
		}
	}

	lastCanvas, lastSidebar := float32(-1), float32(-1)
	for glass := 0.0; glass <= 100; glass++ {
		p := wallpapers.Defaults()
		p.Glass, p.HideSidebarMask = glass, false
		canvas := MaterialAlpha(MaterialCanvas, p, false)
		sidebar := MaterialAlpha(MaterialSidebar, p, false)
		if canvas < lastCanvas || sidebar < lastSidebar {
			t.Fatalf("solidity fell at glass %.0f: canvas %v→%v sidebar %v→%v", glass, lastCanvas, canvas, lastSidebar, sidebar)
		}
		if canvas > float32(maxCanvasSolidity)/100 {
			t.Fatalf("canvas above its cap at glass %.0f: %v", glass, canvas)
		}
		lastCanvas, lastSidebar = canvas, sidebar
	}
}

func TestMaterialAlphaTransparentAndMasks(t *testing.T) {
	p := wallpapers.Defaults()
	p.Glass, p.Terminal = 80, 75
	for _, m := range []Material{MaterialCanvas, MaterialPanel} {
		p.Transparent = false
		if alpha := MaterialAlpha(m, p, false); alpha <= 0 {
			t.Fatalf("material %d is not translucent with a wallpaper: %v", m, alpha)
		}
		p.Transparent = true
		if alpha := MaterialAlpha(m, p, true); alpha != 0 {
			t.Fatalf("transparent theme kept material %d at %v", m, alpha)
		}
	}
	p.Transparent = true
	if alpha := MaterialAlpha(MaterialTerminal, p, false); math.Abs(float64(alpha-.75)) > 1e-4 {
		t.Fatalf("terminal should keep its own slider under a transparent theme: %v", alpha)
	}
	p = wallpapers.Defaults()
	if alpha := MaterialAlpha(MaterialSidebar, p, false); alpha != 0 {
		t.Fatalf("hidden sidebar mask should paint no extra surface: %v", alpha)
	}
	for _, dark := range []bool{false, true} {
		p.Glass = 95
		want := float32(menuSolidityLight)
		if dark {
			want = menuSolidityDark
		}
		if alpha := MaterialAlpha(MaterialMenu, p, dark); alpha != want {
			t.Fatalf("menu alpha dark=%v = %v, want %v", dark, alpha, want)
		}
	}
}

// TestLegacyBackdropAlphasKeepTheirValues pins the fixed-alpha helpers the
// shell already calls, so wiring surfaces to materials cannot change them.
func TestLegacyBackdropAlphasKeepTheirValues(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	base := ui.RGB(20, 20, 20)
	ConfigureBackdrop("wall.png", wallpapers.Defaults())
	if got := BackdropCanvas(base); got.A != base.Alpha(.45).A {
		t.Fatalf("canvas alpha = %d, want 45%%", got.A)
	}
	if got := BackdropSurface(base); got.A != base.Alpha(.8).A {
		t.Fatalf("surface alpha = %d, want 80%%", got.A)
	}
	if got := BackdropSidebar(base); got.A != 0 {
		t.Fatalf("sidebar with its mask hidden = %d, want transparent", got.A)
	}
	hidden := wallpapers.Defaults()
	hidden.HideSidebarMask = false
	ConfigureBackdrop("wall.png", hidden)
	if got := BackdropSidebar(base); got.A != base.Alpha(.625).A {
		t.Fatalf("sidebar alpha = %d, want 62.5%%", got.A)
	}
	terminal := wallpapers.Defaults()
	terminal.Terminal = 40
	ConfigureBackdrop("wall.png", terminal)
	if got := BackdropTerminal(base); got.A != base.Alpha(.4).A {
		t.Fatalf("terminal alpha = %d, want 40%%", got.A)
	}
	ConfigureBackdrop("", wallpapers.Defaults())
	if got := BackdropCanvas(base); got != base && got != base.Alpha(.8) { /* dark show-through keeps the glass share */
		t.Fatal("no backdrop must keep the theme color or its glass share")
	}
}

// gradientWallpaper is a 200×100 image whose red grows left to right, so a
// sample's brightness says where the panel sits.
func gradientWallpaper() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			v := uint8(x * 255 / 199)
			img.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func TestBackdropSamplingFollowsWallpaperBrightness(t *testing.T) {
	sampler := newBackdropSampler(gradientWallpaper(), 1)
	box := ui.Rect{X: 0, Y: 0, W: 200, H: 100}
	panel := func(x float32) ui.Rect { return ui.Rect{X: x, Y: 0, W: 40, H: 100} }
	left, ok := sampler.average(box, panel(0))
	if !ok {
		t.Fatal("a panel over the wallpaper sampled nothing")
	}
	middle, ok := sampler.average(box, panel(80))
	if !ok {
		t.Fatal("middle panel sampled nothing")
	}
	right, ok := sampler.average(box, panel(160))
	if !ok {
		t.Fatal("right panel sampled nothing")
	}
	if !(left.R < middle.R && middle.R < right.R) {
		t.Fatalf("sample brightness does not follow the wallpaper: %d %d %d", left.R, middle.R, right.R)
	}
	if right.R-left.R < 100 {
		t.Fatalf("sampling flattened the wallpaper: %d..%d", left.R, right.R)
	}
	if left.A != 255 {
		t.Fatalf("sampled backdrop must stay opaque: %d", left.A)
	}
}

// TestSurfaceFillTransparencyIsMonotone checks the glass fill approaches its
// panel color as the material gets more opaque, and never leaves the range
// between the backdrop and the tint.
func TestSurfaceFillTransparencyIsMonotone(t *testing.T) {
	sampler := newBackdropSampler(gradientWallpaper(), 1)
	box := ui.Rect{X: 0, Y: 0, W: 200, H: 100}
	panel := ui.Rect{X: 20, Y: 20, W: 40, H: 40}
	backdrop, ok := sampler.average(box, panel)
	if !ok {
		t.Fatal("no backdrop under the panel")
	}
	tint := ui.RGB(240, 240, 240)
	last := float32(-1)
	for _, alpha := range []float32{0, .2, .4, .6, .8, 1} {
		fill := tint.Alpha(alpha).Over(backdrop)
		if alpha > 0 && float32(fill.R) < last {
			t.Fatalf("glass fill fell as it got more opaque at %v: %d after %v", alpha, fill.R, last)
		}
		if fill.R < backdrop.R || fill.R > tint.R {
			t.Fatalf("glass fill left the backdrop..tint range at %v: %d", alpha, fill.R)
		}
		last = float32(fill.R)
	}
}

func TestCoverRegionMapsPanelThroughTheCoverCrop(t *testing.T) {
	// A 200×100 image covers a square window at 2×, so the cover crop drops
	// 100 pixels from each side and the panel sees the middle of the image.
	bounds := image.Rect(0, 0, 200, 100)
	box := ui.Rect{X: 0, Y: 0, W: 200, H: 200}
	region, ok := coverRegion(box, ui.Rect{X: 0, Y: 0, W: 100, H: 100}, bounds)
	if !ok {
		t.Fatal("a panel over the image sampled nothing")
	}
	if want := image.Rect(50, 0, 100, 50); region != want {
		t.Fatalf("cover mapping = %v, want %v", region, want)
	}
	whole, ok := coverRegion(box, box, bounds)
	if !ok || whole != image.Rect(50, 0, 150, 100) {
		t.Fatalf("the whole window sees the crop %v, want (50,0)-(150,100)", whole)
	}
	if _, ok := coverRegion(box, ui.Rect{}, bounds); ok {
		t.Fatal("an empty panel sampled the image")
	}
}

func TestSurfaceFillWithoutBackdropKeepsFixedAlpha(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	ConfigureBackdrop("", wallpapers.Defaults())
	base := ui.RGB(10, 200, 30)
	got := SurfaceFill(ui.Rect{X: 0, Y: 0, W: 10, H: 10}, base, .5)
	if got != base.Alpha(.5) {
		t.Fatalf("fill without a wallpaper = %+v, want the tint at its alpha", got)
	}
	if active := BackdropActive(); active {
		t.Fatal("no wallpaper or gradient reported a backdrop")
	}
}

// pngDataURL builds a wallpaper the shell can decode in memory.
func pngDataURL(t *testing.T, c color.RGBA, w, h int) string {
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

// TestWallpaperLayerDimsAndSamplingUsesTheDim checks the sheet's layer order:
// the image paints first, then the dim mask (bg-mask-1), and surfaces sample
// what that leaves.
func TestWallpaperLayerDimsAndSamplingUsesTheDim(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"

	render := func(mutate func(*wallpapers.Preferences), dark bool) (*ui.Tester, ui.Rect) {
		t.Helper()
		p := prefs
		mutate(&p)
		tt := ui.NewTester(func(c *ui.Context) {
			paint := WallpaperBackgroundPainter(url, p, c.Preferences().ReduceMotion)
			ui.Box(c).Fill().Draw(func(painter *ui.Painter, r ui.Rect) { paint(painter, r) })
		}, 40, 20)
		tt.SetDark(dark)
		tt.Frame()
		return tt, ui.Rect{X: 0, Y: 0, W: 40, H: 20}
	}

	light, box := render(func(*wallpapers.Preferences) {}, false)
	wantLight := 255 * (1 - maskLight)
	if got := float64(light.Image().RGBAAt(20, 10).R); math.Abs(got-wantLight) > 2 {
		t.Fatalf("light wallpaper pixel = %.0f, want %.0f after the dim mask", got, wantLight)
	}
	sample, ok := SampleBackdrop(ui.Rect{X: 8, Y: 4, W: 12, H: 8})
	if !ok {
		t.Fatal("the painted wallpaper did not reach the sampler")
	}
	if math.Abs(float64(sample.R)-wantLight) > 3 {
		t.Fatalf("sampled backdrop = %d, want the dimmed wallpaper %.0f", sample.R, wantLight)
	}

	dark, _ := render(func(*wallpapers.Preferences) {}, true)
	wantDark := 255 * (1 - maskDark)
	if got := float64(dark.Image().RGBAAt(20, 10).R); math.Abs(got-wantDark) > 2 {
		t.Fatalf("dark wallpaper pixel = %.0f, want %.0f after the dim mask", got, wantDark)
	}
	if sample, ok := SampleBackdrop(box); !ok || math.Abs(float64(sample.R)-wantDark) > 3 {
		t.Fatalf("dark sampled backdrop = %v, want %.0f", sample, wantDark)
	}

	transparent, _ := render(func(p *wallpapers.Preferences) { p.Transparent = true }, true)
	if got := float64(transparent.Image().RGBAAt(20, 10).R); got < 253 {
		t.Fatalf("transparent theme dimmed the wallpaper to %.0f", got)
	}
}

// TestGlassPaintsFrostedWallpaper checks that a frosted material keeps the
// wallpaper's local brightness instead of flattening it to one color.
func TestGlassPaintsFrostedWallpaper(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 40, G: 40, B: 40, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		ui.Box(c).Absolute().Left(4).Top(4).Width(16).Height(12).Draw(Glass(ui.RGB(255, 255, 255), MaterialMenu))
	}, 40, 20)
	tt.Frame()
	inside := tt.Image().RGBAAt(10, 8)
	outside := tt.Image().RGBAAt(30, 8)
	if outside.R > 80 {
		t.Fatalf("wallpaper outside the glass = %d, want it dimmed", outside.R)
	}
	if inside.R <= outside.R {
		t.Fatalf("menu glass did not tint its panel: inside %d outside %d", inside.R, outside.R)
	}
}

func TestGlassDrawsNothingUnderTransparentTheme(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 200, G: 200, B: 200, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Transparent = true
	prefs.Effect = "none"
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		ui.Box(c).Absolute().Left(0).Top(0).Width(20).Height(20).Draw(Glass(ui.RGB(0, 0, 0), MaterialPanel))
	}, 40, 20)
	tt.Frame()
	if got := tt.Image().RGBAAt(10, 10).R; got < 195 {
		t.Fatalf("transparent chrome painted over the wallpaper: %d", got)
	}
}

// TestTransparentThemeNeedsAWallpaper keeps the harness gate: the transparent
// theme only clears the chrome while a wallpaper is live, so a profile with
// just the ambient gradient keeps legible surfaces.
func TestTransparentThemeNeedsAWallpaper(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	p := wallpapers.Defaults()
	p.Transparent, p.Effect, p.EffectExplicit = true, "gradient", true
	base := ui.RGB(20, 20, 20)
	ConfigureBackdrop("", p)
	if got := BackdropCanvas(base); got.A != base.Alpha(.45).A {
		t.Fatalf("transparent theme cleared the chrome over a gradient: alpha %d", got.A)
	}
	ConfigureBackdrop("wall.png", p)
	if got := BackdropCanvas(base); got.A != 0 {
		t.Fatalf("transparent theme kept the canvas at alpha %d over a wallpaper", got.A)
	}
}

// TestGlassKeepsRoundedCorners paints a rounded glass card over a wallpaper and
// checks the corners: outside the arc the wallpaper shows, inside it the glass
// tints, and the same fill without a radius would square the corner.
func TestGlassKeepsRoundedCorners(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 40, G: 40, B: 40, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	const radius = 20
	render := func(round float32) *ui.Tester {
		t.Helper()
		tt := ui.NewTester(func(c *ui.Context) {
			paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
			ui.Box(c).Absolute().Left(10).Top(10).Width(60).Height(40).Radius(round).
				Draw(Glass(ui.RGB(255, 255, 255), MaterialMenu, round))
		}, 80, 60)
		tt.Frame()
		return tt
	}
	rounded := render(radius)
	corner := rounded.Image().RGBAAt(10, 10)
	if corner.R > 80 {
		t.Fatalf("rounded glass squared its corner: %d", corner.R)
	}
	// (16,16) is inside the 20 DIP arc whose centre is (30,30).
	inside := rounded.Image().RGBAAt(16, 16)
	if inside.R < 120 {
		t.Fatalf("glass left the rounded corner unpainted: %d", inside.R)
	}
	center := rounded.Image().RGBAAt(40, 30)
	if center.R < 120 {
		t.Fatalf("glass did not fill the panel: %d", center.R)
	}
	square := render(0).Image().RGBAAt(10, 10)
	if square.R < 120 {
		t.Fatalf("the square control did not paint its corner: %d", square.R)
	}
}

// TestMenuSurfaceRoundsTheMenuGlass checks the menu helper threads its radius
// into the material, border and shadow it paints.
func TestMenuSurfaceRoundsTheMenuGlass(t *testing.T) {
	url := pngDataURL(t, color.RGBA{R: 40, G: 40, B: 40, A: 255}, 8, 8)
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		MenuSurface(ui.Box(c).Absolute().Left(20).Top(10).Width(40).Height(30).Radius(10),
			ui.RGB(255, 255, 255), ui.RGB(200, 200, 200), 10)
	}, 80, 60)
	tt.Frame()
	if got := tt.Image().RGBAAt(20, 10); got.R > 90 {
		t.Fatalf("menu corner is not rounded: %v", got)
	}
	if got := tt.Image().RGBAAt(40, 25); got.R < 120 {
		t.Fatalf("menu glass did not paint its face: %v", got)
	}
}

// TestGlassSurfaceReplacesTheSolidBackground checks the wiring helper: it must
// clear an element's own background, or the opaque fill would hide the glass.
func TestGlassSurfaceReplacesTheSolidBackground(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 40, G: 40, B: 40, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		GlassSurface(ui.Box(c).Absolute().Left(0).Top(0).Width(40).Height(20), ui.RGB(255, 255, 255), MaterialCanvas)
	}, 40, 20)
	tt.Frame()
	got := float64(tt.Image().RGBAAt(20, 10).R)
	// Canvas 45% over the dimmed wallpaper, not the opaque fill.
	want := float64(255)*.45 + float64(40)*(1-maskLight)*.55
	if math.Abs(got-want) > 4 {
		t.Fatalf("glass canvas pixel = %.0f, want about %.0f", got, want)
	}
}

func TestMenuSurfacePaintsMaterialAndShadow(t *testing.T) {
	base := ui.RGB(30, 30, 30)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(250, 250, 250))
		panel := MenuSurface(ui.Box(c).Absolute().Left(10).Top(4).Width(20).Height(12).Radius(4), base, ui.RGB(200, 200, 200), 4)
		panel.Label("menu")
	}, 40, 24)
	tt.Frame()
	// Without a wallpaper the menu keeps the theme's own fill.
	if got := tt.Image().RGBAAt(20, 10).R; got != base.R {
		t.Fatalf("menu surface pixel = %d, want the theme fill %d", got, base.R)
	}
	if got := tt.Image().RGBAAt(20, 22).R; got > 250 {
		t.Fatalf("menu surface cast no shadow: %d", got)
	}
}

// TestMenuSurfaceSharesTheWallpaper checks the menu keeps its own share
// (--dsw-menu-surface-fill) once a wallpaper is behind it.
func TestMenuSurfaceSharesTheWallpaper(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	const wall = 200
	url := pngDataURL(t, color.RGBA{R: wall, G: wall, B: wall, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	base := ui.RGB(30, 30, 30)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		MenuSurface(ui.Box(c).Absolute().Left(10).Top(4).Width(30).Height(16).Radius(4), base, ui.RGB(200, 200, 200), 4)
	}, 50, 24)
	tt.Frame()
	under := float64(wall) * (1 - maskDark)
	want := float64(base.R)*menuSolidityDark + under*(1-menuSolidityDark)
	if got := float64(tt.Image().RGBAAt(25, 12).R); math.Abs(got-want) > 4 {
		t.Fatalf("menu over the wallpaper = %.0f, want about %.0f", got, want)
	}
}

func TestMenuShadowAndElevationPaint(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) {
			MenuShadow(p, ui.Rect{X: 8, Y: 6, W: 20, H: 12}, 4)
		})
		panel := ui.Box(c).Absolute().Left(4).Top(2).Width(10).Height(6).Background(ui.RGB(255, 255, 255))
		MenuElevation(panel)
	}, 40, 24)
	tt.Frame()
	if got := tt.Image().RGBAAt(18, 22).R; got > 250 {
		t.Fatalf("menu shadow left the area below the panel untouched: %d", got)
	}
}

// TestGlassPatchKeepsLocalBrightness checks the frost itself: a patch sampled
// over the dark side of the wallpaper stays darker than one over the light
// side, which is what makes a panel follow what is beneath it.
func TestGlassPatchKeepsLocalBrightness(t *testing.T) {
	sampler := newBackdropSampler(gradientWallpaper(), 1)
	box := ui.Rect{X: 0, Y: 0, W: 200, H: 100}
	dark, ok := sampler.patch(box, ui.Rect{X: 0, Y: 0, W: 40, H: 40}, 4, 4, 1)
	if !ok {
		t.Fatal("no frost over the dark side")
	}
	light, ok := sampler.patch(box, ui.Rect{X: 160, Y: 0, W: 40, H: 40}, 4, 4, 1)
	if !ok {
		t.Fatal("no frost over the light side")
	}
	if got, want := dark.RGBAAt(2, 2).R, light.RGBAAt(2, 2).R; got >= want {
		t.Fatalf("frost did not follow the wallpaper: dark %d light %d", got, want)
	}
	if dark.RGBAAt(0, 0).R >= dark.RGBAAt(3, 0).R {
		t.Fatal("frost cells are not ordered along the wallpaper")
	}
	saturated, ok := sampler.patch(box, ui.Rect{X: 160, Y: 0, W: 40, H: 40}, 4, 4, 1.5)
	if !ok {
		t.Fatal("no saturated frost")
	}
	if saturated.RGBAAt(2, 2).A != 255 {
		t.Fatal("frost cells must stay opaque")
	}
}

// TestMetallicSheenPaintsOnlyWhileHovered covers the metallic paint helper the
// custom-painted buttons use.
func TestMetallicSheenPaintsOnlyWhileHovered(t *testing.T) {
	render := func(hovered bool) []byte {
		t.Helper()
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(r, ui.RGB(40, 40, 200), 0)
				MetallicSheen(p, r, 6, hovered, false)
			})
		}, 80, 24)
		tt.Frame()
		return bytes.Clone(tt.Image().Pix)
	}
	plain, hovered := render(false), render(true)
	if bytes.Equal(plain, hovered) {
		t.Fatal("hovered button painted no metallic sheen")
	}
	var changed int
	for i := 0; i+3 < len(plain); i += 4 {
		if plain[i] != hovered[i] || plain[i+1] != hovered[i+1] || plain[i+2] != hovered[i+2] {
			changed++
		}
	}
	if changed < 100 {
		t.Fatalf("metallic sheen covered only %d pixels", changed)
	}
}
