package chrome_test

import (
	"image"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func flatPrefs() wallpapers.Preferences {
	p := wallpapers.Defaults()
	p.Effect = "none" // what wallpaperBackdropPrefs produces for an empty path
	return p
}

func renderFlat(t *testing.T, dark bool, w, h int, paint func(*ui.Painter, ui.Rect)) *image.RGBA {
	t.Helper()
	colors := theme.Resolve(theme.IDDeepseekLight)
	if dark {
		colors = theme.Resolve(theme.IDDeepseekDark)
	}
	active := colors.Active(dark)
	tt := ui.NewTester(func(c *ui.Context) {
		if dark {
			c.SetTheme(ui.DarkTheme())
		} else {
			c.SetTheme(ui.LightTheme())
		}
		ui.Column(c).Fill().Background(ui.Hex(active.BG)).Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
	}, w, h)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(2 * time.Second))
	return tt.Image()
}

// TestNoWallpaperFlatBackdrop probes both themes: with no image the painter
// must add nothing, so the rendered frame is exactly the theme base colour (the
// native profile has no wallpaper and shows the flat base).
func TestNoWallpaperFlatBackdrop(t *testing.T) {
	for _, dark := range []bool{true, false} {
		name := "light"
		if dark {
			name = "dark"
		}
		without := renderFlat(t, dark, 64, 40, func(*ui.Painter, ui.Rect) {})
		with := renderFlat(t, dark, 64, 40, chrome.WallpaperBackgroundPainter("", flatPrefs()))
		if got, want := with.RGBAAt(32, 20), without.RGBAAt(32, 20); got != want {
			t.Fatalf("%s: flat painter changed the base pixel: got %v want %v", name, got, want)
		}
		// Control: the aurora still paints, so the probe above is meaningful.
		gradient := wallpapers.Defaults()
		gradient.Effect = "gradient"
		gradient.EffectExplicit = true
		gradient.Preset = "dune"
		if got := renderFlat(t, dark, 64, 40, chrome.WallpaperBackgroundPainter("", gradient)).RGBAAt(32, 20); got == with.RGBAAt(32, 20) {
			t.Fatalf("%s: aurora control did not paint", name)
		}
	}
}

// TestNoWallpaperFlatSnapshot exports the MyGo dark 1440×860 no-wallpaper frame
// for the D-MAIN comparison. Set NO_WALLPAPER_SNAPSHOT_DIR to write it.
func TestNoWallpaperFlatSnapshot(t *testing.T) {
	dir := os.Getenv("NO_WALLPAPER_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set NO_WALLPAPER_SNAPSHOT_DIR to export the flat no-wallpaper capture")
	}
	img := renderFlat(t, true, 1440, 860, chrome.WallpaperBackgroundPainter("", flatPrefs()))
	saveSnapshot(t, dir, "round216-no-wallpaper-dark-flat", img)
}
