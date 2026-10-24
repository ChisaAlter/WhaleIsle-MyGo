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

// TestPresetDoesNotPaintWithoutEffect: preset colours and a variant alone must
// never paint the gradient (round102 saw a first-run profile keep a seeded
// preset and read as aurora); only Effect "gradient" does.
func TestPresetDoesNotPaintWithoutEffect(t *testing.T) {
	colors := theme.Resolve(theme.IDDeepseekDark).Active(true)
	base := ui.Hex(colors.BG)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Preset = "dune"
	prefs.Colors = wallpapers.PresetColors("dune")
	render := func(paint func(*ui.Painter, ui.Rect)) *image.RGBA {
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.DarkTheme())
			ui.Column(c).Fill().Background(base).Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		}, 64, 40)
		start := time.Unix(1000, 0)
		tt.SetTime(start)
		tt.SetTime(start.Add(2 * time.Second))
		return tt.Image()
	}
	without := render(func(*ui.Painter, ui.Rect) {})
	with := render(chrome.WallpaperBackgroundPainter("", prefs))
	if got, want := with.RGBAAt(32, 20), without.RGBAAt(32, 20); got != want {
		t.Fatalf("preset-only prefs painted: got %v want %v", got, want)
	}
	// Control: an explicit gradient with the same preset does paint.
	prefs.Effect, prefs.EffectExplicit = "gradient", true
	if got := render(chrome.WallpaperBackgroundPainter("", prefs)).RGBAAt(32, 20); got == with.RGBAAt(32, 20) {
		t.Fatal("explicit gradient control did not paint")
	}
}

// TestFlatMaskedCanvasSnapshot exports the dark 1440×860 canvas under the
// native bg-mask for the nine-point sample. Set FLAT_CANVAS_SNAPSHOT_DIR.
func TestFlatMaskedCanvasSnapshot(t *testing.T) {
	dir := os.Getenv("FLAT_CANVAS_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set FLAT_CANVAS_SNAPSHOT_DIR to export the masked flat canvas")
	}
	colors := theme.Resolve(theme.IDDeepseekDark).Active(true)
	base := chrome.FlatBackdropFill(ui.Hex(colors.BG), true)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Column(c).Fill().Background(base)
	}, 1440, 860)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(2 * time.Second))
	saveSnapshot(t, dir, "round229-flat-backdrop", tt.Image())
}
