package chrome_test

import (
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
)

// TestWallpaperEffectSnapshot renders the dark seeded wallpaper at the audit
// size with the native effect fields, so the blur/pixelate mapping can be
// compared against the installed CSS-implied look. It is an export seam, like
// the other dark snapshots: set WALLPAPER_EFFECT_SNAPSHOT_DIR to write the PNGs.
func TestWallpaperEffectSnapshot(t *testing.T) {
	dir := os.Getenv("WALLPAPER_EFFECT_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set WALLPAPER_EFFECT_SNAPSHOT_DIR to export wallpaper effect captures")
	}
	w, h := 1440, 860
	for _, tc := range []struct {
		name     string
		blur     float64
		pixelate float64
	}{
		{"round203-wallpaper-dark-blur0-pixel0", 0, 0},
		{"round203-wallpaper-dark-blur8-pixel12", 8, 12},
	} {
		prefs := wallpapers.Defaults()
		prefs.Effect = "none"
		prefs.Blur, prefs.Pixelate = tc.blur, tc.pixelate
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.DarkTheme())
			paint := chrome.WallpaperBackgroundPainter(darkWallpaperDataURL(t), prefs, c.Preferences().ReduceMotion)
			ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		}, w, h)
		start := time.Unix(1000, 0)
		tt.SetTime(start)
		tt.SetTime(start.Add(2 * time.Second))
		saveSnapshot(t, dir, tc.name, tt.Image())
	}
}
