//go:build windows

package d3d11

import (
	"image"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/scene"
)

var (
	_ gpu.IdleRenderer      = (*Renderer)(nil)
	_ gpu.TrimmableRenderer = (*Renderer)(nil)
)

func TestImageCacheLRUByteBudget(t *testing.T) {
	r := &Renderer{
		images:      map[uint64]*texture{},
		imageBudget: 1024, // 2 × 512 bytes
	}
	add := func(id uint64, frame uint64, bytes int64) {
		r.frame = frame
		r.imageClock++
		r.images[id] = &texture{bytes: bytes, lastFrame: frame, lastUse: r.imageClock}
		r.imageBytes += bytes
		r.evictImages(r.effectiveImageBudget(), false)
	}
	touch := func(id uint64, frame uint64) {
		r.frame = frame
		r.imageClock++
		t := r.images[id]
		t.lastFrame = frame
		t.lastUse = r.imageClock
	}

	// Frame 1: insert img 1 and img 2 (total 1024 B == budget).
	add(1, 1, 512)
	add(2, 1, 512)
	if len(r.images) != 2 || r.imageBytes != 1024 {
		t.Fatalf("frame 1: len=%d bytes=%d, want 2, 1024", len(r.images), r.imageBytes)
	}

	// Frame 2: touch img 1 so img 2 becomes LRU, then insert img 3 -> img 2 evicted.
	touch(1, 2)
	add(3, 2, 512)
	if _, ok := r.images[2]; ok {
		t.Fatal("frame 2: expected LRU image 2 to be evicted")
	}
	if _, ok := r.images[1]; !ok {
		t.Fatal("frame 2: recently used image 1 was evicted")
	}
	if _, ok := r.images[3]; !ok {
		t.Fatal("frame 2: newly added image 3 was evicted")
	}
	if r.imageBytes != 1024 {
		t.Fatalf("frame 2: imageBytes=%d, want 1024", r.imageBytes)
	}

	// Frame 3: inserting two 512B images in the same frame preserves current-frame
	// images while evicting older-frame images (1 and 3).
	add(4, 3, 512)
	add(5, 3, 512)
	add(6, 3, 512) // over budget, but all 3 belong to current frame 3
	if len(r.images) != 3 || r.imageBytes != 1536 {
		t.Fatalf("frame 3 in-flight: len=%d bytes=%d, want 3, 1536 (must not evict current frame)", len(r.images), r.imageBytes)
	}

	// Idle() evicts stale images and enforces budget on remaining ones.
	r.Idle()
	if r.imageBytes > r.effectiveImageBudget() || len(r.images) != 2 {
		t.Fatalf("after Idle: len=%d bytes=%d, want 2, <=1024", len(r.images), r.imageBytes)
	}
	if _, ok := r.images[4]; ok {
		t.Fatal("after Idle: expected oldest image 4 to be evicted")
	}

	// Trim() clears all cached images.
	r.Trim()
	if len(r.images) != 0 || r.imageBytes != 0 {
		t.Fatalf("after Trim: len=%d bytes=%d, want 0, 0", len(r.images), r.imageBytes)
	}
}

func TestShouldShrinkTarget(t *testing.T) {
	if !shouldShrinkTarget(2000, 1000, 800, 400) {
		t.Fatal("expected significant shrink (2000x1000 -> 800x400) to trigger")
	}
	if shouldShrinkTarget(1000, 500, 800, 400) {
		t.Fatal("should not shrink when current target matches 25% headroom")
	}
	if shouldShrinkTarget(200, 200, 50, 50) {
		t.Fatal("should not churn small targets under minShrinkPixels")
	}
	if shouldShrinkTarget(0, 0, 400, 400) || shouldShrinkTarget(1000, 1000, 0, 0) {
		t.Fatal("zero dimensions must not trigger shrink")
	}
}

func TestD3D11IdleTrimAndBackdropShrink(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	r, err := New(hiddenWindow(t, 400, 300))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()

	// Verify byte-budget LRU eviction on real D3D11 textures.
	r.imageBudget = 64 * 64 * 4 * 2 // room for 2 64×64 RGBA images
	makeImg := func(fill byte) *scene.Image {
		pix := make([]byte, 64*64*4)
		for i := range pix {
			pix[i] = fill
		}
		return scene.NewImage(&image.RGBA{Pix: pix, Stride: 64 * 4, Rect: image.Rect(0, 0, 64, 64)})
	}
	img1, img2, img3 := makeImg(10), makeImg(20), makeImg(30)

	renderWith := func(imgs ...*scene.Image) {
		t.Helper()
		s := &scene.Scene{Width: 400, Height: 300}
		for i, im := range imgs {
			s.Ops = append(s.Ops, scene.Op{
				Kind:  scene.OpImage,
				Rect:  scene.Rect{X: float32(i * 64), Y: 0, W: 64, H: 64},
				Src:   scene.Rect{W: 64, H: 64},
				Image: im,
			})
		}
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
	}

	renderWith(img1, img2)
	if len(r.images) != 2 || r.imageBytes != 64*64*4*2 {
		t.Fatalf("after frame 1: len=%d bytes=%d", len(r.images), r.imageBytes)
	}
	renderWith(img1, img3)
	if len(r.images) != 2 || r.imageBytes != 64*64*4*2 {
		t.Fatalf("after frame 2 LRU eviction: len=%d bytes=%d, want 2 and %d", len(r.images), r.imageBytes, 64*64*4*2)
	}
	if _, ok := r.images[img2.ID()]; ok {
		t.Fatal("expected img2 to be evicted on budget pressure")
	}

	// Verify grab/backdrop shrink when backdrop shrinks significantly.
	r.b.Backdrops = []scene.Backdrop{{Area: image.Rect(0, 0, 1200, 800), Down: 1}}
	if err := r.fitBackdrop(); err != nil {
		t.Fatal(err)
	}
	bigGrabW, bigGrabH := r.grab.w, r.grab.h
	if bigGrabW < 1200 || bigGrabH < 800 {
		t.Fatalf("big grab = %dx%d, want >= 1200x800", bigGrabW, bigGrabH)
	}
	r.b.Backdrops = []scene.Backdrop{{Area: image.Rect(0, 0, 400, 300), Down: 1}}
	if err := r.fitBackdrop(); err != nil {
		t.Fatal(err)
	}
	if r.grab.w >= bigGrabW || r.grab.h >= bigGrabH {
		t.Fatalf("grab did not shrink: got %dx%d, was %dx%d", r.grab.w, r.grab.h, bigGrabW, bigGrabH)
	}

	// Verify Idle() reclaims scratch backdrop textures and Trim() clears images,
	// and a subsequent Render() succeeds cleanly.
	r.Idle()
	if r.grab.tex != 0 || r.backdrop[0].tex != 0 || r.backdrop[1].tex != 0 {
		t.Fatal("Idle() did not release grab/backdrop textures")
	}
	r.Trim()
	if len(r.images) != 0 || r.imageBytes != 0 || r.instBuf != 0 {
		t.Fatalf("Trim() left resources: images=%d bytes=%d instBuf=%v", len(r.images), r.imageBytes, r.instBuf)
	}
	renderWith(img3)
	if len(r.images) != 1 || r.imageBytes != 64*64*4 {
		t.Fatalf("render after Trim(): images=%d bytes=%d", len(r.images), r.imageBytes)
	}
}
