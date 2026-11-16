package settings

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
)

func testAvatarPNGBytes(t *testing.T, r uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, color.RGBA{R: r, G: 80, B: 160, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func resetSettingsAvatarCacheForTest(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		accountImages.Lock()
		accountImages.cache = nil
		accountImages.entries = nil
		accountImages.order = nil
		accountImages.inFlight = nil
		accountImages.Unlock()
	})
}

func TestAccountAvatarLoadsAsyncCachesNegativeAndBoundsLRU(t *testing.T) {
	resetSettingsAvatarCacheForTest(t)

	cache := imgcache.New(t.TempDir())
	SetAccountImageCache(cache)

	validURL := "https://example.com/avatar-valid.png"
	if err := os.WriteFile(cache.FilePath(validURL), testAvatarPNGBytes(t, 42), 0600); err != nil {
		t.Fatal(err)
	}

	var firstResult *ui.Bitmap
	tt := ui.NewTester(func(c *ui.Context) {
		firstResult = accountAvatar(c, validURL)
	}, 120, 60)
	if firstResult != nil {
		t.Fatal("expected first cached-file decode to happen off UI render thread")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if firstResult != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if firstResult == nil {
		t.Fatal("async accountAvatar decode did not complete")
	}

	// Corrupt bytes on disk must be negatively cached so repeat frames do not loop spawning reads.
	corruptURL := "https://example.com/avatar-corrupt.png"
	if err := os.WriteFile(cache.FilePath(corruptURL), []byte("not-a-png"), 0600); err != nil {
		t.Fatal(err)
	}
	_ = accountAvatar(nil, corruptURL)
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		accountImages.Lock()
		entry, ok := accountImages.entries[corruptURL]
		inFlight := accountImages.inFlight[corruptURL]
		accountImages.Unlock()
		if ok && !inFlight && entry.bmp == nil && !entry.failedAt.IsZero() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	_ = accountAvatar(nil, corruptURL)
	accountImages.Lock()
	restarted := accountImages.inFlight[corruptURL]
	accountImages.Unlock()
	if restarted {
		t.Fatal("negatively cached avatar re-dispatched read on subsequent frame")
	}

	// Retry clears the negative entry cleanly.
	RetryAccountAvatar(corruptURL, nil)
	accountImages.Lock()
	_, stillCached := accountImages.entries[corruptURL]
	accountImages.Unlock()
	if stillCached {
		t.Fatal("RetryAccountAvatar did not clear negative entry")
	}

	// Eviction caps entries at maxSettingsAvatarCache.
	for i := 0; i < maxSettingsAvatarCache+6; i++ {
		u := fmt.Sprintf("https://example.com/u-%d.png", i)
		if err := os.WriteFile(cache.FilePath(u), testAvatarPNGBytes(t, uint8(i)), 0600); err != nil {
			t.Fatal(err)
		}
		_ = accountAvatar(nil, u)
		wait := time.Now().Add(3 * time.Second)
		for time.Now().Before(wait) {
			accountImages.Lock()
			_, done := accountImages.entries[u]
			accountImages.Unlock()
			if done {
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
	accountImages.Lock()
	count := len(accountImages.entries)
	accountImages.Unlock()
	if count > maxSettingsAvatarCache {
		t.Fatalf("settings avatar LRU exceeded bound: got %d, want <= %d", count, maxSettingsAvatarCache)
	}
}
