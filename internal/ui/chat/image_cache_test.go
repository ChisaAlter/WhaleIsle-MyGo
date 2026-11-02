package chat

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func resetChatImageCache(t *testing.T) {
	t.Helper()
	clear := func() {
		chatImageCache.mu.Lock()
		chatImageCache.entries, chatImageCache.bytes = nil, 0
		chatImageCache.mu.Unlock()
	}
	clear()
	t.Cleanup(clear)
}

func writeTestPNG(t *testing.T, dir, name string, shade uint8) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{shade, 0, 0, 255})
		}
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

// Two images drawn in one frame used to evict each other from the single-slot
// cache, so both were read and decoded again every frame.
func TestLoadChatBitmapKeepsSeveralImages(t *testing.T) {
	resetChatImageCache(t)
	dir := t.TempDir()
	a := writeTestPNG(t, dir, "a.png", 10)
	b := writeTestPNG(t, dir, "b.png", 200)
	bmpA, bmpB := loadChatBitmap(a), loadChatBitmap(b)
	if bmpA == nil || bmpB == nil || bmpA == bmpB {
		t.Fatal("images did not load")
	}
	for i := 0; i < 5; i++ {
		if loadChatBitmap(a) != bmpA || loadChatBitmap(b) != bmpB {
			t.Fatalf("frame %d decoded an image again instead of reusing the cached bitmap", i)
		}
	}
	// The cache returns what was decoded, not a copy that a file change replaces.
	if err := os.Remove(a); err != nil {
		t.Fatal(err)
	}
	if loadChatBitmap(a) != bmpA {
		t.Fatal("a cached image must not be re-read from disk")
	}
}

func TestLoadChatBitmapEvictsLeastRecentlyUsed(t *testing.T) {
	resetChatImageCache(t)
	dir := t.TempDir()
	first := writeTestPNG(t, dir, "first.png", 1)
	keep := writeTestPNG(t, dir, "keep.png", 2)
	firstBmp, keepBmp := loadChatBitmap(first), loadChatBitmap(keep)
	for i := 0; i < chatImageCacheEntries; i++ {
		loadChatBitmap(writeTestPNG(t, dir, "n"+strconv.Itoa(i)+".png", uint8(i+10)))
		if loadChatBitmap(keep) != keepBmp { // keep stays recently used
			t.Fatalf("recently used image was evicted after %d others", i+1)
		}
	}
	chatImageCache.mu.Lock()
	n := len(chatImageCache.entries)
	chatImageCache.mu.Unlock()
	if n > chatImageCacheEntries {
		t.Fatalf("cache holds %d entries, bound is %d", n, chatImageCacheEntries)
	}
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if got := loadChatBitmap(first); got == firstBmp || got != nil {
		t.Fatal("the least recently used image was not evicted")
	}
}

func TestLoadChatBitmapRemembersFailures(t *testing.T) {
	resetChatImageCache(t)
	missing := filepath.Join(t.TempDir(), "gone.png")
	if loadChatBitmap(missing) != nil {
		t.Fatal("a missing file produced a bitmap")
	}
	// The file appearing later is only noticed once the entry ages out; what
	// matters per frame is that a failure does not hit the disk again.
	writeTestPNG(t, filepath.Dir(missing), "gone.png", 5)
	if loadChatBitmap(missing) != nil {
		t.Fatal("a remembered failure re-read the file")
	}
	chatImageCache.mu.Lock()
	defer chatImageCache.mu.Unlock()
	if len(chatImageCache.entries) != 1 || chatImageCache.entries[0].bmp != nil || chatImageCache.bytes != 0 {
		t.Fatalf("unexpected cache state: %+v bytes=%d", chatImageCache.entries, chatImageCache.bytes)
	}
}
