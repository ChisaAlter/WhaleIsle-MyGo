package chrome

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/wallpapers"
)

func solidTestWallpaperDataURL(t *testing.T, c color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestPrewarmWallpaperTracksDecodeState(t *testing.T) {
	svc := media.New(media.Config{Workers: -1})
	prevMedia, prevSettle := wallpaperMedia, wallpaperSettle
	wallpaperMedia = func() *media.Service { return svc }
	wallpaperSettle = nil
	t.Cleanup(func() {
		wallpaperMedia, wallpaperSettle = prevMedia, prevSettle
		ConfigureBackdrop("", wallpapers.Defaults())
	})

	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Blur = 0
	prefs.Pixelate = 0
	prefs.Transparent = true

	if !PrewarmWallpaper("", prefs, 200, 120) || !PrewarmWallpaper("   ", prefs, 200, 120) {
		t.Fatal("empty wallpaper path must be immediately ready")
	}

	url := solidTestWallpaperDataURL(t, color.RGBA{R: 40, G: 110, B: 230, A: 255})
	if PrewarmWallpaper(url, prefs, 200, 120) {
		t.Fatal("PrewarmWallpaper returned true while decode was still pending")
	}
	if svc.Queued() != 1 {
		t.Fatalf("queued jobs = %d, want 1", svc.Queued())
	}

	svc.RunPending()
	if !PrewarmWallpaper(url, prefs, 200, 120) {
		t.Fatal("PrewarmWallpaper returned false after decode finished")
	}
	decodesBeforePaint := svc.Decodes()

	// The very first frame painted after PrewarmWallpaper settles must hit the
	// warmed bitmap cache and draw wallpaper pixels immediately.
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		paint := WallpaperBackgroundPainter(url, prefs, true)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
	}, 200, 120)
	tt.SetTime(time.Unix(1000, 0))
	px := tt.Image().RGBAAt(100, 60)
	if px.B < 200 || px.R > 80 {
		t.Fatalf("first frame did not paint prewarmed wallpaper pixels: got %+v", px)
	}
	if svc.Decodes() != decodesBeforePaint {
		t.Fatalf("painting prewarmed wallpaper triggered extra decode: before=%d after=%d", decodesBeforePaint, svc.Decodes())
	}

	// Failed decodes must also unblock handoff once settled.
	corrupt := "data:image/png;base64,bm90LWEtcG5n"
	if PrewarmWallpaper(corrupt, prefs, 200, 120) {
		t.Fatal("corrupt wallpaper reported ready before decode ran")
	}
	svc.RunPending()
	if !PrewarmWallpaper(corrupt, prefs, 200, 120) {
		t.Fatal("failed wallpaper decode did not unblock PrewarmWallpaper")
	}
}
