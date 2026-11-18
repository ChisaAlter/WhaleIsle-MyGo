package chrome

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func encodeTestPNG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func resetWallpaperCachesForTest(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		wallpaperCache.mu.Lock()
		wallpaperCache.entries = nil
		wallpaperCache.order = nil
		wallpaperCache.inFlight = nil
		wallpaperCache.mu.Unlock()

		decodedWallpaperCache.mu.Lock()
		decodedWallpaperCache.entries = nil
		decodedWallpaperCache.order = nil
		decodedWallpaperCache.inFlight = nil
		decodedWallpaperCache.mu.Unlock()
	})
}

func TestWallpaperBackdropLoadsAsyncCachesNegativeAndEvictsBounded(t *testing.T) {
	resetWallpaperCachesForTest(t)

	dir := t.TempDir()
	goodPath := filepath.Join(dir, "good.png")
	if err := os.WriteFile(goodPath, encodeTestPNG(t, color.RGBA{R: 12, G: 34, B: 56, A: 255}), 0600); err != nil {
		t.Fatal(err)
	}

	var invalidations atomic.Int32
	done := make(chan struct{}, 1)
	if bmp := loadWallpaperBitmapAsync(goodPath, func() {
		invalidations.Add(1)
		select {
		case done <- struct{}{}:
		default:
		}
	}); bmp != nil {
		t.Fatal("first load returned bitmap synchronously instead of loading asynchronously")
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("async wallpaper load did not invoke Invalidate")
	}
	if bmp := loadWallpaperBitmapAsync(goodPath, nil); bmp == nil {
		t.Fatal("cached bitmap missing after async invalidation")
	}

	// Negative entry for missing path is stored and prevents repeated async spawns within TTL.
	missingPath := filepath.Join(dir, "missing.png")
	if bmp := loadWallpaperBitmapAsync(missingPath, nil); bmp != nil {
		t.Fatal("expected nil for missing file")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		wallpaperCache.mu.Lock()
		entry, ok := wallpaperCache.entries[missingPath]
		inFlight := wallpaperCache.inFlight[missingPath]
		wallpaperCache.mu.Unlock()
		if ok && !inFlight && entry.bmp == nil && !entry.failedAt.IsZero() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	wallpaperCache.mu.Lock()
	entry, ok := wallpaperCache.entries[missingPath]
	inFlight := wallpaperCache.inFlight[missingPath]
	wallpaperCache.mu.Unlock()
	if !ok || inFlight || entry.bmp != nil || entry.failedAt.IsZero() {
		t.Fatalf("missing path was not negatively cached cleanly: ok=%v inFlight=%v entry=%+v", ok, inFlight, entry)
	}
	// Second lookup must hit negative cache without re-arming inFlight.
	_ = loadWallpaperBitmapAsync(missingPath, nil)
	wallpaperCache.mu.Lock()
	restarted := wallpaperCache.inFlight[missingPath]
	wallpaperCache.mu.Unlock()
	if restarted {
		t.Fatal("negative cache entry restarted background read on immediate lookup")
	}

	// Eviction keeps cache bounded by maxWallpaperCacheEntries.
	for i := 0; i < maxWallpaperCacheEntries+4; i++ {
		p := filepath.Join(dir, fmt.Sprintf("wall-%d.png", i))
		if err := os.WriteFile(p, encodeTestPNG(t, color.RGBA{R: uint8(i), A: 255}), 0600); err != nil {
			t.Fatal(err)
		}
		ready := make(chan struct{}, 1)
		_ = loadWallpaperBitmapAsync(p, func() {
			select {
			case ready <- struct{}{}:
			default:
			}
		})
		select {
		case <-ready:
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out loading wallpaper %d", i)
		}
	}
	wallpaperCache.mu.Lock()
	count := len(wallpaperCache.entries)
	wallpaperCache.mu.Unlock()
	if count > maxWallpaperCacheEntries {
		t.Fatalf("wallpaper cache grew to %d entries, want <= %d", count, maxWallpaperCacheEntries)
	}
}

func TestDecodedWallpaperDiskIsAsyncAndNegativeCached(t *testing.T) {
	resetWallpaperCachesForTest(t)

	dir := t.TempDir()
	goodPath := filepath.Join(dir, "screen.png")
	if err := os.WriteFile(goodPath, encodeTestPNG(t, color.RGBA{R: 200, G: 100, B: 50, A: 255}), 0600); err != nil {
		t.Fatal(err)
	}
	if rgba := decodedWallpaper(goodPath); rgba != nil {
		t.Fatal("disk wallpaper decoded synchronously on first lookup")
	}
	deadline := time.Now().Add(3 * time.Second)
	var got *image.RGBA
	for time.Now().Before(deadline) {
		if got = decodedWallpaper(goodPath); got != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got == nil {
		t.Fatal("async decodedWallpaper never populated cache")
	}

	missingPath := filepath.Join(dir, "absent.png")
	_ = decodedWallpaper(missingPath)
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !decodedWallpaperInFlight(missingPath) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	decodedWallpaperCache.mu.Lock()
	neg, ok := decodedWallpaperCache.entries[missingPath]
	decodedWallpaperCache.mu.Unlock()
	if !ok || neg.rgba != nil || neg.failedAt.IsZero() {
		t.Fatalf("missing disk wallpaper was not negatively cached: ok=%v entry=%+v", ok, neg)
	}
	// Repeat lookup stays synchronous miss (no inFlight spawn).
	_ = decodedWallpaper(missingPath)
	if decodedWallpaperInFlight(missingPath) {
		t.Fatal("negative cache missed and spawned duplicate async load")
	}
}
