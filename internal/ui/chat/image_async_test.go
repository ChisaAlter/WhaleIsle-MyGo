package chat

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

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func resetImageAsyncTestCaches() {
	chatImageCache.mu.Lock()
	chatImageCache.entries = nil
	chatImageCache.bytes = 0
	chatImageCache.inFlight = nil
	chatImageCache.mu.Unlock()

	localImagePathCache.mu.Lock()
	localImagePathCache.entries = nil
	localImagePathCache.inFlight = nil
	localImagePathCache.mu.Unlock()
}

func writeSolidPNG(t *testing.T, dir, name string, r uint8) string {
	t.Helper()
	p := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: r, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLocalImagePathAsyncSingleflightNegativeCacheAndLRU(t *testing.T) {
	resetImageAsyncTestCaches()
	t.Cleanup(resetImageAsyncTestCaches)

	dir := t.TempDir()
	imgPath := writeSolidPNG(t, dir, "ok.png", 10)
	missingPath := filepath.Join(dir, "missing.png")

	var statCalls atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	oldStat := localImageStatFn
	localImageStatFn = func(name string) (os.FileInfo, error) {
		if name == imgPath {
			if statCalls.Add(1) == 1 {
				close(entered)
				<-release
			}
		} else {
			statCalls.Add(1)
		}
		return oldStat(name)
	}
	t.Cleanup(func() { localImageStatFn = oldStat })

	invalidated := make(chan struct{}, 4)
	inv := func() {
		select {
		case invalidated <- struct{}{}:
		default:
		}
	}

	// Repeated UI-thread lookups while in flight never block and deduplicate into one stat.
	for range 4 {
		got, pending := cachedLocalImagePath(imgPath, inv)
		if got != "" || !pending {
			t.Fatalf("expected non-blocking pending on first lookup, got (%q, %v)", got, pending)
		}
	}

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("background stat did not start")
	}
	if n := statCalls.Load(); n != 1 {
		t.Fatalf("singleflight allowed %d concurrent stats for same path", n)
	}

	close(release)
	select {
	case <-invalidated:
	case <-time.After(2 * time.Second):
		t.Fatal("completion did not invoke Invalidate")
	}

	got, pending := cachedLocalImagePath(imgPath, nil)
	if got != imgPath || pending {
		t.Fatalf("expected cached path %q ready, got (%q, %v)", imgPath, got, pending)
	}
	if n := statCalls.Load(); n != 1 {
		t.Fatalf("cached lookup triggered extra stat: %d", n)
	}

	// Missing paths enter negative cache so later renders avoid re-statting.
	missingDone := make(chan struct{}, 2)
	got, pending = cachedLocalImagePath(missingPath, func() {
		select {
		case missingDone <- struct{}{}:
		default:
		}
	})
	if !pending || got != "" {
		t.Fatalf("expected pending for missing lookup, got (%q, %v)", got, pending)
	}
	select {
	case <-missingDone:
	case <-time.After(2 * time.Second):
		t.Fatal("missing stat did not invoke Invalidate")
	}
	for range 3 {
		got, pending = cachedLocalImagePath(missingPath, nil)
		if got != "" || pending {
			t.Fatalf("expected cached negative miss, got (%q, %v)", got, pending)
		}
	}
	if n := statCalls.Load(); n != 2 {
		t.Fatalf("negative cache failed to suppress repeat stats: %d", n)
	}

	// Exceeding localImagePathCacheEntries evicts oldest entries cleanly.
	for i := 0; i < localImagePathCacheEntries+20; i++ {
		p := filepath.Join(dir, fmt.Sprintf("missing-%03d.png", i))
		cachedLocalImagePath(p, nil)
		localImagePath(p) // wait synchronously via shared singleflight to populate LRU
	}
	localImagePathCache.mu.Lock()
	nEntries := len(localImagePathCache.entries)
	localImagePathCache.mu.Unlock()
	if nEntries > localImagePathCacheEntries {
		t.Fatalf("localImagePathCache exceeded cap: %d > %d", nEntries, localImagePathCacheEntries)
	}
}

func TestImageInlineViewRenderNeverBlocksAndDecodesOffMutex(t *testing.T) {
	resetImageAsyncTestCaches()
	t.Cleanup(resetImageAsyncTestCaches)

	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(oldLocale) })

	dir := t.TempDir()
	slowPNG := writeSolidPNG(t, dir, "slow.png", 42)
	fastPNG := writeSolidPNG(t, dir, "fast.png", 99)

	// Pre-warm local path cache so ImageInlineView enters paintInlineImage directly.
	if got := localImagePath(slowPNG); got != slowPNG {
		t.Fatalf("prewarm localImagePath failed: %q", got)
	}

	var slowReads atomic.Int32
	readStarted := make(chan struct{})
	readRelease := make(chan struct{})
	oldRead := chatImageReadFileFn
	chatImageReadFileFn = func(path string) ([]byte, error) {
		if path == slowPNG {
			if slowReads.Add(1) == 1 {
				close(readStarted)
				<-readRelease
			}
		}
		return oldRead(path)
	}
	t.Cleanup(func() { chatImageReadFileFn = oldRead })

	// First render must complete immediately without blocking on slowPNG read/decode.
	renderDone := make(chan *ui.Tester, 1)
	go func() {
		tt := ui.NewTester(func(c *ui.Context) {
			ImageInlineView(c, slowPNG, ui.RGB(0, 0, 0), ui.RGB(100, 100, 100), nil)
		}, 320, 240)
		renderDone <- tt
	}()

	var tt *ui.Tester
	select {
	case tt = <-renderDone:
	case <-time.After(2 * time.Second):
		t.Fatal("ImageInlineView blocked UI render frame on disk read/decode")
	}

	if !tt.HasText(i18n.T(i18n.KeyChatImageLoading)) {
		t.Fatalf("expected loading placeholder while async decode is in flight, texts=%v", tt.Texts())
	}

	select {
	case <-readStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("background bitmap read did not start")
	}

	// While slowPNG is blocked reading/decoding, fastPNG must still load cleanly
	// proving chatImageCache.mu is not held across disk I/O or ui.DecodeBitmap.
	fastDone := make(chan *ui.Bitmap, 1)
	go func() {
		fastDone <- loadChatBitmap(fastPNG)
	}()
	select {
	case bmp := <-fastDone:
		if bmp == nil {
			t.Fatal("concurrent fastPNG decode returned nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("unrelated image load stalled while slowPNG read/decoded (mutex held across I/O)")
	}

	// Duplicate render/cachedChatBitmap calls while in flight share singleflight.
	bmp, pending := cachedChatBitmap(slowPNG, nil)
	if bmp != nil || !pending {
		t.Fatalf("expected still pending before release, got (%v, %v)", bmp, pending)
	}
	if slowReads.Load() != 1 {
		t.Fatalf("duplicate reads started for slowPNG: %d", slowReads.Load())
	}

	close(readRelease)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if !tt.HasText(i18n.T(i18n.KeyChatImageLoading)) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if tt.HasText(i18n.T(i18n.KeyChatImageLoading)) {
		t.Fatalf("expected image view to replace loading placeholder once decoded; texts=%v", tt.Texts())
	}
	if slowReads.Load() != 1 {
		t.Fatalf("expected singleflight count 1, got %d", slowReads.Load())
	}
}
