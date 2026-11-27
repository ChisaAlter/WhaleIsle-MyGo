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

type countingFramePacer struct {
	handle    uintptr
	tokens    int
	acquired  bool
	tryCalls  int
	markCalls int
}

func (p *countingFramePacer) WaitHandle() uintptr { return p.handle }
func (p *countingFramePacer) TryAcquireFrame() bool {
	p.tryCalls++
	if p.acquired {
		return true
	}
	if p.tokens > 0 {
		p.tokens--
		p.acquired = true
		return true
	}
	return false
}
func (p *countingFramePacer) MarkFrameReady() {
	p.markCalls++
	p.acquired = true
}

func TestSurfaceBypassPathsConsumeReadyToken(t *testing.T) {
	b := New()
	b.inMainLoop = true
	w := &window{b: b}
	s := &surface{w: w}
	p := &countingFramePacer{handle: 0x5678, tokens: 1}
	s.SetFramePacer(p)

	cases := []struct {
		name  string
		setup func()
		clean func()
	}{
		{"immediatePaint", func() { s.immediatePaint = true }, func() { s.immediatePaint = false }},
		{"deferredForce", func() { s.deferredForce = true }, func() { s.deferredForce = false }},
		{"inSizeMove", func() { w.inSizeMove = true }, func() { w.inSizeMove = false }},
		{"inMenu", func() { w.inMenu = true }, func() { w.inMenu = false }},
		{"outsideMainLoop", func() { b.inMainLoop = false }, func() { b.inMainLoop = true }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p.tokens = 1
			p.acquired = false
			prevCalls := p.tryCalls

			tc.setup()
			deferred := s.shouldDeferPaint()
			tc.clean()

			if deferred {
				t.Fatalf("%s: should not defer paint", tc.name)
			}
			if p.tryCalls == prevCalls {
				t.Fatalf("%s: bypass path skipped TryAcquireFrame", tc.name)
			}
			if !p.acquired || p.tokens != 0 {
				t.Fatalf("%s: expected ready token consumed (acquired=%v tokens=%d)", tc.name, p.acquired, p.tokens)
			}
			// Simulate Present completing and releasing 1 token back.
			p.acquired = false
			p.tokens++
			if p.tokens != 1 {
				t.Fatalf("%s: token count drifted to %d, want 1", tc.name, p.tokens)
			}
		})
	}
}
