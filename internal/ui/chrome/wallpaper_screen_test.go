package chrome_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
)

// gradientWallpaper builds a normalized wallpaper: a smooth ramp with a tiny
// ripple, so every pixelated block takes a distinct centre colour and adjacent
// blocks never merge into one run.
func gradientWallpaper(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x * 255 / max(1, w-1)),
				G: uint8(y * 255 / max(1, h-1)),
				B: uint8((x*7 + y*3) % 256),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// runMode returns the most common run length of identical pixels along a row.
func runMode(img image.Image, y int) int {
	b := img.Bounds()
	counts := map[int]int{}
	run := 1
	var prev color.Color
	if b.Dx() > 0 {
		prev = img.At(b.Min.X, y)
	}
	for x := b.Min.X + 1; x < b.Max.X; x++ {
		c := img.At(x, y)
		if c == prev {
			run++
			continue
		}
		counts[run]++
		run, prev = 1, c
	}
	counts[run]++
	best, bestCount := 0, -1
	for length, n := range counts {
		if n > bestCount || (n == bestCount && length > best) {
			best, bestCount = length, n
		}
	}
	return best
}

// TestWallpaperScreenBlockSize renders the same normalized wallpaper from a
// 320×200 and a 1440×860 source at pixelate factor 3 and probes the rendered
// image: the on-screen block must be the same 3px in both, because pixelation
// now happens after cover-fit at the window size.
func TestWallpaperScreenBlockSize(t *testing.T) {
	const (
		w, h = 300, 200
		// Factor 3 = 1 + percent/100*19.
		percent = 200.0 / 19
	)
	var modes []int
	for _, size := range [][2]int{{320, 200}, {1440, 860}} {
		prefs := wallpapers.Defaults()
		prefs.Effect = "none"
		prefs.Pixelate = percent
		prefs.Blur = 0
		url := gradientWallpaper(t, size[0], size[1])
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.DarkTheme())
			paint := chrome.WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
			ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		}, w, h)
		start := time.Unix(1000, 0)
		tt.SetTime(start)
		tt.SetTime(start.Add(2 * time.Second))
		modes = append(modes, runMode(tt.Image(), h/2))
	}
	if modes[0] != 3 || modes[1] != 3 {
		t.Fatalf("screen block sizes differ across sources: 320×200 → %dpx, 1440×860 → %dpx (want 3 both)", modes[0], modes[1])
	}
}
