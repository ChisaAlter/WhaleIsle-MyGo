package market

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/ui/i18n"
)

func TestMarketScreenshotPublicHTTPSFilterAndLimit(t *testing.T) {
	values := []string{"http://owned.invalid/plain.png", "/relative.png", "file:///owned.png", "https://user:password@owned.invalid/private.png", "not a url", "HTTPS://owned.invalid/first.png"}
	for i := 0; i < 15; i++ {
		values = append(values, fmt.Sprintf("https://owned.invalid/%d.png", i))
	}
	images := marketScreenshots(values)
	if len(images) != 12 || images[0] != "https://owned.invalid/first.png" || images[11] != "https://owned.invalid/10.png" {
		t.Fatalf("gallery admitted a non-public source or changed the source limit: %v", images)
	}
}

func TestMarketScreenshotGalleryInteractionFailureAndSourceReset(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	readmeRemoteImages.RLock()
	previous := readmeRemoteImages.cache
	readmeRemoteImages.RUnlock()
	cache := imgcache.New(t.TempDir())
	SetRemoteImageCache(cache)
	t.Cleanup(func() { SetRemoteImageCache(previous) })
	first, failed, replacement := "https://owned.invalid/first.png", "https://owned.invalid/failed.png", "https://owned.invalid/replacement.png"
	canvas := image.NewRGBA(image.Rect(0, 0, 32, 16))
	canvas.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		t.Fatal(err)
	}
	// Seed only this test's cache, so drawing never needs a network transport.
	for _, source := range []string{first, replacement} {
		if err := os.WriteFile(cache.FilePath(source), encoded.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(cache.FilePath(failed), []byte("invalid image"), 0600); err != nil {
		t.Fatal(err)
	}
	item := desktopmarket.CatalogItem{ID: "owned/plugin", Screenshots: []string{first, failed}}
	var opened []string
	model := Model{OnOpenURL: func(source string) { opened = append(opened, source) }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			marketScreenshotGallery(c, model, item, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface)
		})
	}, 800, 700)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	box, ok := tt.Find("Plugin screenshot 1")
	if !ok || math.Abs(float64(box.W/box.H)-1.6) > .01 || !tt.HasText("1 / 2") {
		t.Fatalf("first screenshot did not use the original 16:10 slot: %v texts=%v", box, tt.Texts())
	}
	click("Previous image")
	if !tt.HasText("1 / 2") {
		t.Fatal("Previous moved before the first image")
	}
	click("Next image")
	if !tt.HasText("2 / 2") || !tt.HasText("Image could not be loaded.") {
		t.Fatalf("failed image did not retain its slot and navigation: %v", tt.Texts())
	}
	click("Next image")
	click("Open image")
	if len(opened) != 1 || opened[0] != failed || !tt.HasText("2 / 2") {
		t.Fatalf("last image boundary or explicit source link changed: %v", opened)
	}
	click("Previous image")
	if !tt.HasText("1 / 2") || tt.HasText("Image could not be loaded.") {
		t.Fatal("failed image status leaked onto a decoded screenshot")
	}
	item.Screenshots = []string{replacement}
	tt.Frame()
	if !tt.HasText("1 / 1") {
		t.Fatal("changed screenshot sources retained the prior selection")
	}
	click("Open image")
	if len(opened) != 2 || opened[1] != replacement {
		t.Fatalf("replacement gallery opened a stale source: %v", opened)
	}
	item.Screenshots = nil
	tt.Frame()
	if !tt.HasText("No screenshots available.") {
		t.Fatal("empty gallery did not show the original empty state")
	}
}
