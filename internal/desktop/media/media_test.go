package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/svgrender"
)

func pngBytes(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range w * h {
		img.Set(i%w, i/w, c)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRequestDedupesAndSharesOneBitmap(t *testing.T) {
	s := New(Config{Workers: -1})
	o := s.NewOwner(nil, nil)
	data := pngBytes(t, 4, 4, color.White)
	key, src := Bytes(data, "")
	a := s.Request(o, key, src)
	b := s.Request(o, key, src)
	if a != b || s.Queued() != 1 {
		t.Fatalf("duplicate request queued twice: same=%v queued=%d", a == b, s.Queued())
	}
	if a.State() != Pending {
		t.Fatal("request decoded synchronously")
	}
	s.RunPending()
	other := s.NewOwner(nil, nil)
	c := s.Request(other, key, src)
	if a.State() != Ready || c.Bitmap() == nil || c.Bitmap() != a.Bitmap() {
		t.Fatal("a second owner did not share the decoded bitmap")
	}
	if s.Decodes() != 1 {
		t.Fatalf("decodes = %d, want 1", s.Decodes())
	}
}

func TestLRUStaysWithinBudget(t *testing.T) {
	s := New(Config{Workers: -1, MemoryBudget: 3 * 64 * 64 * 4})
	o := s.NewOwner(nil, nil)
	var handles []*Handle
	for i := range 6 {
		key, src := Bytes(pngBytes(t, 64, 64, color.RGBA{uint8(i), 0, 0, 255}), "")
		handles = append(handles, s.Request(o, key, src))
		s.RunPending()
	}
	if got := s.MemoryBytes(); got > 3*64*64*4 {
		t.Fatalf("LRU holds %d bytes, over budget", got)
	}
	// The newest pictures survive; the oldest was evicted and decodes again.
	key, src := Bytes(pngBytes(t, 64, 64, color.RGBA{0, 0, 0, 255}), "")
	if s.Request(o, key, src).State() != Pending {
		t.Fatal("evicted picture still cached")
	}
	key, src = Bytes(pngBytes(t, 64, 64, color.RGBA{5, 0, 0, 255}), "")
	if s.Request(o, key, src) != handles[5] {
		t.Fatal("most recent picture was evicted")
	}
}

func TestReleaseDropsQueuedWork(t *testing.T) {
	s := New(Config{Workers: -1})
	o := s.NewOwner(nil, nil)
	var loads atomic.Int32
	h := s.Request(o, Key{ID: "x"}, Source{Load: func(context.Context) ([]byte, error) {
		loads.Add(1)
		return pngBytes(t, 1, 1, color.White), nil
	}})
	shared := s.Request(o, Key{ID: "y"}, Source{Load: func(context.Context) ([]byte, error) {
		loads.Add(1)
		return pngBytes(t, 1, 1, color.White), nil
	}})
	keeper := s.NewOwner(nil, nil)
	s.Request(keeper, Key{ID: "y"}, Source{})
	o.Release()
	if n := s.RunPending(); n != 1 || loads.Load() != 1 {
		t.Fatalf("ran %d jobs with %d loads; dropped work must never run", n, loads.Load())
	}
	if h.State() != Pending || shared.State() != Ready {
		t.Fatalf("released-only job state %v, shared job state %v", h.State(), shared.State())
	}
	if s.Request(o, Key{ID: "z"}, Source{Bytes: []byte("x")}).Failure() != FailCanceled {
		t.Fatal("a released owner started new work")
	}
}

func TestContextEndCancelsRunningWork(t *testing.T) {
	s := New(Config{Workers: 1})
	ctx, cancel := context.WithCancel(context.Background())
	notified := make(chan struct{}, 1)
	o := s.NewOwner(ctx, func() { notified <- struct{}{} })
	started := make(chan struct{})
	h := s.Request(o, Key{ID: "slow"}, Source{Load: func(ctx context.Context) ([]byte, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}})
	<-started
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for !o.Released() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	if h.State() != Pending {
		t.Fatalf("cancelled work published a result: %v", h.State())
	}
	select {
	case <-notified:
		t.Fatal("a cancelled owner was notified")
	default:
	}
}

func TestCompletionNotifiesOwner(t *testing.T) {
	s := New(Config{})
	done := make(chan struct{}, 4)
	o := s.NewOwner(nil, func() { done <- struct{}{} })
	key, src := Bytes(pngBytes(t, 2, 2, color.White), "")
	h := s.Request(o, key, src)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("no completion notification")
	}
	if h.State() != Ready {
		t.Fatal("notified before ready")
	}
}

const filterSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><filter id="f"><feGaussianBlur stdDeviation="1"/></filter><rect width="16" height="16" fill="#08f" filter="url(#f)"/></svg>`

func TestRasterWithoutTextLoadsNoFonts(t *testing.T) {
	s := New(Config{Workers: -1})
	o := s.NewOwner(nil, nil)
	before := svgrender.FontLoads()
	key, src := Bytes([]byte(filterSVG), ".svg")
	h := s.Request(o, key, src)
	s.RunPending()
	if h.State() != Ready || h.Bitmap() == nil {
		t.Fatalf("filter SVG not rasterised: %v", h.Failure())
	}
	if got := svgrender.FontLoads() - before; got != 0 {
		t.Fatalf("text-free raster loaded fonts %d times", got)
	}
}

func TestNativeVectorNeedsNoRaster(t *testing.T) {
	data, err := os.ReadFile("testdata/plugin-icons/auto-review.svg")
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{Workers: -1})
	key, src := Bytes(data, ".svg")
	h := s.Request(nil, key, src)
	s.RunPending()
	p, ok := h.Picture()
	if !ok || p.SVG == nil || p.Bitmap != nil || s.Rasters() != 0 {
		t.Fatalf("plugin icon did not take the native vector path: %+v rasters=%d", p, s.Rasters())
	}
}

func TestTextRasterLoadsFontsOncePerWorker(t *testing.T) {
	s := New(Config{Workers: -1})
	before := svgrender.FontLoads()
	for _, text := range []string{"One", "Two"} {
		key, src := Bytes([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="60" height="20"><text x="1" y="15" font-family="Go">`+text+`</text></svg>`), ".svg")
		s.Request(nil, key, src)
	}
	s.RunPending()
	if got := svgrender.FontLoads() - before; got != 1 {
		t.Fatalf("font set loaded %d times for two text SVGs on one worker, want 1", got)
	}
}

func TestDiskCacheHitAfterRestart(t *testing.T) {
	dir := t.TempDir()
	first := New(Config{Workers: -1, CacheDir: dir})
	key, src := Bytes([]byte(filterSVG), ".svg")
	first.Request(nil, key, src)
	first.RunPending()
	if first.Rasters() != 1 {
		t.Fatalf("first run rasters = %d", first.Rasters())
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*", "*.png"))
	if len(files) != 1 {
		t.Fatalf("disk cache files = %d, want 1", len(files))
	}
	second := New(Config{Workers: -1, CacheDir: dir})
	h := second.Request(nil, key, src)
	second.RunPending()
	if h.State() != Ready || second.Rasters() != 0 {
		t.Fatalf("restart re-rasterised: state=%v rasters=%d", h.State(), second.Rasters())
	}
}

func TestDiskCachePrunesToBudget(t *testing.T) {
	dir := t.TempDir()
	d := newDiskCache(dir, 4000)
	for i := range 10 {
		name := []byte("aa00000000000000")
		name[2] = byte('0' + i)
		d.put(string(name), bytes.Repeat([]byte{1}, 900))
		old := time.Now().Add(time.Duration(i-20) * time.Minute)
		_ = os.Chtimes(d.path(string(name)), old, old)
	}
	d.prune()
	files, _ := filepath.Glob(filepath.Join(dir, "*", "*.png"))
	if len(files) == 0 || int64(len(files))*900 > 4000 {
		t.Fatalf("pruned cache holds %d files", len(files))
	}
	if _, err := os.Stat(d.path("aa90000000000000")); err != nil {
		t.Fatal("the most recent file was pruned")
	}
}

func TestAnimationUsesOneBitmap(t *testing.T) {
	// A two-frame GIF.
	var buf bytes.Buffer
	if err := encodeTwoFrameGIF(&buf); err != nil {
		t.Fatal(err)
	}
	s := New(Config{Workers: -1})
	key, src := Bytes(buf.Bytes(), "")
	h := s.Request(nil, key, src)
	s.RunPending()
	if !h.Animated() {
		t.Fatal("GIF is not animated")
	}
	start := time.Now()
	b0, wait := h.Frame(start)
	b1, _ := h.Frame(start.Add(wait + time.Millisecond))
	if b0 == nil || b0 != b1 {
		t.Fatal("animation frames are separate bitmaps")
	}
}

func TestFileAutoRevRefreshesChangedFile(t *testing.T) {
	s := New(Config{Workers: -1})
	o := s.NewOwner(nil, nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "wall.png")
	if err := os.WriteFile(path, pngBytes(t, 4, 4, color.White), 0o600); err != nil {
		t.Fatal(err)
	}
	key, src := File(path, "")
	h := s.Request(o, key, src)
	s.RunPending()
	first, ok := h.Picture()
	if !ok || first.Bitmap == nil {
		t.Fatal("first decode not ready")
	}

	// Same path, new bytes: the stat (size+mtime) changes.
	if err := os.WriteFile(path, pngBytes(t, 8, 4, color.Black), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, time.Now(), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	stale := s.Request(o, key, src)
	if stale.Bitmap() != first.Bitmap {
		t.Fatal("the cache-hit frame should still serve the previous picture")
	}
	s.RunPending()
	next := s.Request(o, key, src)
	s.RunPending()
	got, ok := next.Picture()
	if !ok || got.Bitmap == nil {
		t.Fatal("second decode not ready")
	}
	if got.Bitmap == first.Bitmap {
		t.Fatal("changed file still served the stale bitmap")
	}
	if s.Decodes() != 2 {
		t.Fatalf("decodes = %d", s.Decodes())
	}
}

func TestFileRevisionKeysDistinctPictures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wall.png")
	if err := os.WriteFile(path, pngBytes(t, 4, 4, color.White), 0o600); err != nil {
		t.Fatal(err)
	}
	ka, _ := File(path, "1:100")
	kb, _ := File(path, "2:200")
	if ka.ID == kb.ID {
		t.Fatal("different revisions must not share a key")
	}
	if kb2, _ := File(path, "2:200"); kb2.ID != kb.ID {
		t.Fatal("same revision must share the key")
	}
}
