package chrome

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
)

// The harness's transparent theme keeps every chrome surface at 0% fill
// (TRANSPARENT_GLASS_SOLIDITY) and drops the wallpaper's dim mask, so the image
// shows through unmasked. Hairlines and text are separate paints and stay: a
// panel with no fill would otherwise disappear into the wallpaper.
func TestTransparentThemeKeepsHairlinesAndText(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 180, G: 180, B: 180, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Transparent = true
	prefs.Effect = "none"
	border := ui.RGBA(255, 255, 255, .28)
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		// A chrome surface: no fill under the transparent theme, but it still
		// paints its hairline and its text.
		panel := ui.Box(c).Absolute().Left(10).Top(10).Width(60).Height(40).Radius(8).Clip().
			Background(ui.Transparent).Border(.5, border)
		panel.Draw(Glass(ui.RGBA(255, 255, 255, .9), MaterialPanel, 8))
		panel.DrawOver(func(p *ui.Painter, r ui.Rect) { p.Stroke(r, border, 8, .5) })
		panel.Children(func() {
			ui.Text(c, "Readable text").FontSize(11).TextColor(ui.RGB(20, 20, 20))
		})
	}, 90, 60)
	tt.Frame()
	image := tt.Image()
	outside := image.RGBAAt(80, 50)
	inside := image.RGBAAt(40, 30)
	edge := image.RGBAAt(40, 10)
	if outside.R != 180 {
		t.Fatalf("wallpaper = %d, want the unmasked 180", outside.R)
	}
	if inside.R != outside.R || inside.G != outside.G || inside.B != outside.B {
		t.Fatalf("transparent chrome tinted the panel: %v on wallpaper %v", inside, outside)
	}
	if RelativeLuminance(pixel(edge)) <= RelativeLuminance(pixel(inside)) {
		t.Fatalf("hairline vanished with the fill: %v on %v", edge, inside)
	}
	text, ok := tt.Find("Readable text")
	if !ok {
		t.Fatal("text disappeared with the fill")
	}
	ink := 0
	for y := int(text.Y); y < int(text.Y+text.H); y++ {
		for x := int(text.X); x < int(text.X+text.W); x++ {
			if image.RGBAAt(x, y).R < 120 {
				ink++
			}
		}
	}
	if ink == 0 {
		t.Fatal("text drew no ink over the transparent surface")
	}
}

// TestTransparentThemeKeepsTheWells checks the surfaces the harness does not
// zero: the terminal keeps its own terminal-opacity slider and menus keep
// --dsw-menu-surface-fill, so they still paint over a transparent theme.
func TestTransparentThemeKeepsTheWells(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 200, G: 200, B: 200, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Transparent, prefs.Effect = true, "none"
	prefs.Terminal = 75
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		ui.Box(c).Absolute().Left(0).Top(0).Width(20).Height(20).Draw(Glass(ui.RGB(30, 30, 30), MaterialTerminal))
		ui.Box(c).Absolute().Left(24).Top(0).Width(20).Height(20).Draw(Glass(ui.RGB(30, 30, 30), MaterialMenu))
	}, 50, 20)
	tt.Frame()
	terminal := tt.Image().RGBAAt(10, 10).R
	menu := tt.Image().RGBAAt(34, 10).R
	wallpaper := tt.Image().RGBAAt(46, 10).R
	if terminal >= wallpaper-20 {
		t.Fatalf("terminal well = %d, want it painted over the %d wallpaper", terminal, wallpaper)
	}
	if menu >= wallpaper-20 {
		t.Fatalf("menu surface = %d, want its own fill over the %d wallpaper", menu, wallpaper)
	}
}

// TestTransparentThemeLiftsASharpWallpaper covers TRANSPARENT_MIN_BLUR: with 0%
// surfaces a sharp image would sit right under the text, so the harness raises
// the blur to 20 when the theme becomes effective, and keeps it while it stays
// on.
func TestTransparentThemeLiftsASharpWallpaper(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	url := pngDataURL(t, color.RGBA{R: 0, G: 0, B: 0, A: 255}, 16, 16)
	prefs := wallpapers.Defaults()
	prefs.Transparent, prefs.Effect = true, "none"
	prefs.Blur = 0
	tt := ui.NewTester(func(c *ui.Context) {
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
	}, 40, 40)
	tt.Frame()
	if got := backgroundState.prefs.Blur; got < transparentMinBlur {
		t.Fatalf("transparent theme kept a sharp wallpaper: blur %v", got)
	}
	// A second configure with the saved (sharp) preference keeps the lift for as
	// long as the theme is effective.
	ConfigureBackdrop(url, prefs)
	if got := backgroundState.prefs.Blur; got != transparentMinBlur {
		t.Fatalf("lift did not hold: blur %v", got)
	}
	// Turning it off restores the user's own blur.
	prefs.Transparent = false
	ConfigureBackdrop(url, prefs)
	if got := backgroundState.prefs.Blur; got != 0 {
		t.Fatalf("transparent off kept the lift: blur %v", got)
	}
	// Over the ambient gradient the theme is not effective (no wallpaper), so a
	// sharp look stays the user's call.
	transparent := prefs
	transparent.Transparent, transparent.Effect = true, "gradient"
	ConfigureBackdrop("", transparent)
	if got := backgroundState.prefs.Blur; got != 0 {
		t.Fatalf("the theme lifted the blur without a wallpaper: %v", got)
	}
}
