package svgimage

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/previewprocess"
)

type fakeSVGSession struct {
	mu           sync.Mutex
	done         chan struct{}
	closeOnce    sync.Once
	presentCalls atomic.Int32
	hideCalls    atomic.Int32
	closeCalls   atomic.Int32
}

func newFakeSVGSession() *fakeSVGSession {
	return &fakeSVGSession{done: make(chan struct{})}
}

func (f *fakeSVGSession) Done() <-chan struct{} { return f.done }

func (f *fakeSVGSession) SetTheme(appearance string, vars map[string]string) error { return nil }

func (f *fakeSVGSession) PresentImageSurface(parent uintptr, x, y, w, h, clipX, clipY, clipW, clipH int) error {
	if w == 0 && h == 0 {
		f.hideCalls.Add(1)
	}
	return nil
}

func (f *fakeSVGSession) PresentImageSurfaceClipped(parent uintptr, x, y, w, h, clipX, clipY, clipW, clipH int, shape previewprocess.ImageSurfaceClip) error {
	f.presentCalls.Add(1)
	return nil
}

func (f *fakeSVGSession) PaintImagePixels(p *ui.Painter, r ui.Rect) bool { return false }

func (f *fakeSVGSession) ImageSize() (float64, float64) { return 120, 80 }

func (f *fakeSVGSession) Close() error {
	f.closeOnce.Do(func() {
		f.closeCalls.Add(1)
		close(f.done)
	})
	return nil
}

func waitSVGCond(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestSurfaceSettledIdleZeroHzAndOffscreenReclaim(t *testing.T) {
	sess := newFakeSVGSession()
	validCalls := atomic.Int32{}
	surf := newSurfaceForSession(context.Background(), nil, sess, 40*time.Millisecond, func() bool {
		validCalls.Add(1)
		return true
	})
	t.Cleanup(func() { _ = surf.Close() })

	// Visible settled paint must not spawn periodic repaints or 10 Hz valid polls.
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(120, 80).Draw(func(p *ui.Painter, r ui.Rect) {
			if err := surf.Paint(p, r, "light", "#ffffff"); err != nil {
				t.Errorf("Paint failed: %v", err)
			}
		})
	}, 200, 120)
	_ = tt.Image()

	callsAfterInit := validCalls.Load()
	presentsAfterInit := sess.presentCalls.Load()
	time.Sleep(250 * time.Millisecond)

	if validCalls.Load() != callsAfterInit {
		t.Fatalf("settled visible surface polled valid() %d times while idle", validCalls.Load()-callsAfterInit)
	}
	if sess.presentCalls.Load() != presentsAfterInit {
		t.Fatalf("settled visible surface triggered %d unscheduled Present calls while idle", sess.presentCalls.Load()-presentsAfterInit)
	}
	if sess.closeCalls.Load() != 0 || surf.Reclaimed() {
		t.Fatal("settled visible surface was reclaimed unexpectedly")
	}

	// Hiding once hides immediately without duplicate hide IPC, and re-showing before TTL cancels reclaim.
	surf.Hide()
	surf.Hide()
	if got := sess.hideCalls.Load(); got != 1 {
		t.Fatalf("expected 1 hide IPC for consecutive Hide(), got %d", got)
	}
	time.Sleep(15 * time.Millisecond)
	tt.Frame() // Back to visible before 40ms TTL.
	time.Sleep(55 * time.Millisecond)
	if sess.closeCalls.Load() != 0 || surf.Reclaimed() {
		t.Fatal("surface reclaimed despite becoming visible before TTL")
	}

	// Staying offscreen past TTL reclaims the surface cleanly without an error.
	surf.Hide()
	waitSVGCond(t, time.Second, func() bool {
		return surf.Reclaimed() && sess.closeCalls.Load() == 1
	}, "offscreen surface was not reclaimed after TTL")
	if err := surf.Error(); err != nil {
		t.Fatalf("offscreen reclaim should not set error, got %v", err)
	}
}

func TestSeatSynchronouslyRevokesOwnerAndClearsReclaimedSurfaceForReload(t *testing.T) {
	svgBytes := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="80" viewBox="0 0 120 80"><rect width="120" height="80"/></svg>`)
	seat, err := NewSeat(svgBytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(seat.Close)

	sess := newFakeSVGSession()
	surf := newSurfaceForSession(context.Background(), nil, sess, 25*time.Millisecond, func() bool { return true })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seat.mu.Lock()
	seat.ownerID = "owner-1"
	seat.surface = surf
	seat.ctx = ctx
	seat.cancel = cancel
	seat.attempted = true
	seat.mu.Unlock()

	var ownerValid atomic.Bool
	ownerValid.Store(true)
	owner := &Owner{
		ID:      "owner-1",
		Window:  &mygo.Window{},
		Context: context.Background(),
		Valid:   ownerValid.Load,
	}

	reloaded := atomic.Int32{}
	seat.openFn = func(ctx context.Context, owner *Owner, width, height float32, appearance string, valid func() bool) (*Surface, error) {
		n := reloaded.Add(1)
		_ = n
		return newSurfaceForSession(ctx, nil, newFakeSVGSession(), time.Second, valid), nil
	}

	// Hide via clipped bounds and wait for offscreen reclaim.
	showSVG := true
	restarted := atomic.Int32{}
	tt := ui.NewTester(func(c *ui.Context) {
		seat.Prepare(c, owner)
		if showSVG {
			ui.Box(c).Size(120, 80).Draw(func(p *ui.Painter, r ui.Rect) {
				seat.Paint(p, r, ui.Contain)
			})
		}
	}, 200, 120)

	// Scroll / clip out of bounds so Seat paints !visible -> Surface.Hide().
	seat.mu.Lock()
	seat.notify = func() { restarted.Add(1) }
	seat.mu.Unlock()

	surf.Hide()
	waitSVGCond(t, time.Second, func() bool {
		return surf.Reclaimed() && sess.closeCalls.Load() == 1
	}, "seat surface did not reclaim while hidden")

	// Next Prepare clears the reclaimed surface without latching an error and reloads on demand.
	tt.Frame()
	if err := seat.Error(); err != nil {
		t.Fatalf("seat recorded error on reclaimed surface: %v", err)
	}
	waitSVGCond(t, time.Second, func() bool {
		return reloaded.Load() == 1 && seat.LoadStatus().Loaded
	}, "reclaimed seat did not reload cleanly on return")

	// Install another active surface, revoke owner synchronously, and verify immediate teardown on Prepare.
	sess2 := newFakeSVGSession()
	surf2 := newSurfaceForSession(context.Background(), nil, sess2, time.Second, func() bool { return true })
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	seat.mu.Lock()
	seat.surface = surf2
	seat.ctx = ctx2
	seat.cancel = cancel2
	seat.err = nil
	seat.mu.Unlock()

	ownerValid.Store(false)
	tt.Frame()

	st := seat.LoadStatus()
	if st.Loaded || st.Pending || !st.Retired {
		t.Fatalf("owner revocation did not synchronously retire seat: %+v", st)
	}
	waitSVGCond(t, time.Second, func() bool {
		return sess2.closeCalls.Load() == 1
	}, "revoked surface session was not closed")
}
