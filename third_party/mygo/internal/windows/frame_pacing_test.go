//go:build windows && (amd64 || arm64)

package windows

import "testing"

type fakeFramePacer struct {
	handle uintptr
	ready  bool
	marked int
}

func (f *fakeFramePacer) WaitHandle() uintptr { return f.handle }
func (f *fakeFramePacer) TryAcquireFrame() bool {
	if f.ready {
		f.ready = false
		return true
	}
	return false
}
func (f *fakeFramePacer) MarkFrameReady() {
	f.ready = true
	f.marked++
}

func TestSurfaceShouldDeferPaint(t *testing.T) {
	b := New()
	w := &window{b: b}
	s := &surface{w: w}
	pacer := &fakeFramePacer{handle: 0x1234, ready: false}
	s.SetFramePacer(pacer)

	// Outside main loop (e.g. unit test or startup): never defer.
	if s.shouldDeferPaint() {
		t.Fatal("should not defer when not in main loop")
	}

	b.inMainLoop = true

	// Inside main loop while swap chain slot is busy: defer paint so input stays responsive.
	if !s.shouldDeferPaint() {
		t.Fatal("expected paint to defer when TryAcquireFrame returns false")
	}
	if !s.frameDeferred || s.deferredSince.IsZero() {
		t.Fatalf("expected frameDeferred=true and non-zero timestamp, got %v %v", s.frameDeferred, s.deferredSince)
	}

	// Modal loops (menu / live drag) and immediate resize paints bypass deferral.
	w.inMenu = true
	if s.shouldDeferPaint() {
		t.Fatal("should not defer while in menu loop")
	}
	w.inMenu = false

	w.inSizeMove = true
	if s.shouldDeferPaint() {
		t.Fatal("should not defer while in live size/move modal loop")
	}
	w.inSizeMove = false

	s.immediatePaint = true
	if s.shouldDeferPaint() {
		t.Fatal("should not defer during immediate resize redraw")
	}
	s.immediatePaint = false

	// Once the waitable handle signals (MarkFrameReady), paint proceeds immediately.
	pacer.MarkFrameReady()
	if s.shouldDeferPaint() {
		t.Fatal("should not defer after MarkFrameReady")
	}
	if s.frameDeferred {
		t.Fatal("frameDeferred should be cleared when slot is acquired")
	}

	// Detaching pacer clears deferred state.
	s.shouldDeferPaint()
	s.SetFramePacer(nil)
	if s.frameDeferred || s.pacer != nil {
		t.Fatal("SetFramePacer(nil) did not clear deferred state")
	}
}
