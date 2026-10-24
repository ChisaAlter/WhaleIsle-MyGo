package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// stoppedOverlay reports, when the supervisor reaches Stopped, whether the
// closing layer was already marked.
func stoppedOverlay(s *mainState) (*harness.Supervisor, <-chan bool) {
	seen := make(chan bool, 1)
	sup := harness.New(harness.Options{OnChange: func(st harness.Status) {
		if st.State != harness.Stopped {
			return
		}
		s.mu.Lock()
		closing := s.closingOverlay
		s.mu.Unlock()
		select {
		case seen <- closing:
		default:
		}
	}})
	return sup, seen
}

// TestClosingOverlayIsUpBeforeHarnessStop pins the native quit order
// (reference/electron/src/main/index.js:1434-1437): the closing layer is marked
// before the local Harness service stops. No window here is the tray/headless
// case, where the quit still runs without a surface to paint.
func TestClosingOverlayIsUpBeforeHarnessStop(t *testing.T) {
	s := &mainState{}
	sup, seen := stoppedOverlay(s)
	s.stopWithClosingOverlay(nil, sup)
	select {
	case closing := <-seen:
		if !closing {
			t.Fatal("the closing layer must be marked before the local Harness service stops")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the supervisor never reported Stopped")
	}
	s.mu.Lock()
	closing := s.closingOverlay
	s.mu.Unlock()
	if !closing {
		t.Fatal("a quit must leave the closing layer marked for the window's next frame")
	}
}

// TestClosingOverlayPaintDeadlineDoesNotHoldShutdown covers a suspended UI
// thread: the frame never arrives, the wait ends at the native 500ms-style
// deadline, and the shutdown step still runs.
func TestClosingOverlayPaintDeadlineDoesNotHoldShutdown(t *testing.T) {
	s := &mainState{}
	release := make(chan struct{})
	defer close(release)
	// Never calls done: the window's UI thread stopped building frames.
	stalled := func(done func()) { <-release }

	start := time.Now()
	if painted := s.showClosingOverlayForQuit(stalled, 40*time.Millisecond); painted {
		t.Fatal("a frame that never arrives must not report a paint")
	}
	if waited := time.Since(start); waited < 40*time.Millisecond || waited > 2*time.Second {
		t.Fatalf("the paint deadline must bound the wait; waited %v", waited)
	}

	sup, seen := stoppedOverlay(s)
	overlayThenStop(func() {
		s.showClosingOverlayForQuit(stalled, 40*time.Millisecond)
	}, func() {
		sup.Stop()
	})
	select {
	case <-seen:
	case <-time.After(2 * time.Second):
		t.Fatal("a paint that never arrives held the shutdown")
	}
}

// TestClosingOverlayFrameRequestHandlesAMissingWindow covers the failed path of
// the paint request itself: no window, no budget, no claim of a paint.
func TestClosingOverlayFrameRequestHandlesAMissingWindow(t *testing.T) {
	if request := closingOverlayFrameRequest(nil); request != nil {
		t.Fatal("a nil window must not post a paint")
	}
	if waitForClosingFrame(closingOverlayFrameRequest(nil), time.Second) {
		t.Fatal("a closed window must not report a painted frame")
	}
	if waitForClosingFrame(nil, time.Second) {
		t.Fatal("no request must not report a painted frame")
	}
	if !waitForClosingFrame(func(done func()) { done() }, 50*time.Millisecond) {
		t.Fatal("a frame that arrives must report its paint")
	}
}

// TestPaintClosingOverlayLayerFollowsTheTheme proves the window view paints the
// frame's palette and localized copy while a quit is in flight, and paints
// nothing once no quit is in flight.
func TestPaintClosingOverlayLayerFollowsTheTheme(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	const page = 200
	pageRed := ui.RGB(page, 30, 30)
	darkPalette := theme.ResolveWithSystem(theme.IDDeepseekDark, true).Active(true)
	lightPalette := theme.ResolveWithSystem(theme.IDDeepseekLight, false).Active(false)
	render := func(s *mainState, colors theme.Colors) *ui.Tester {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Background(pageRed)
			s.paintClosingOverlayLayer(c, colors)
		}, 400, 300)
		tt.Frame()
		return tt
	}

	tt := render(&mainState{closingOverlay: true}, darkPalette)
	if !tt.HasText(i18n.Lookup(i18n.LocaleEN, i18n.KeyClosingTitle)) {
		t.Fatalf("the closing layer is missing its title; texts = %q", tt.Texts())
	}
	bg := theme.Color(darkPalette.BG)
	if got := tt.Image().RGBAAt(1, 1); got.R != bg.R || got.G != bg.G || got.B != bg.B {
		t.Fatalf("background %v, want the dark theme %v", got, bg)
	}
	light := render(&mainState{closingOverlay: true}, lightPalette)
	lightBG := theme.Color(lightPalette.BG)
	if got := light.Image().RGBAAt(1, 1); got.R != lightBG.R || got.G != lightBG.G || got.B != lightBG.B {
		t.Fatalf("background %v, want the light theme %v", got, lightBG)
	}
	if darkBG := theme.Color(darkPalette.BG); lightBG == darkBG {
		t.Fatal("the two palettes must differ for this test to mean anything")
	}

	idle := render(&mainState{}, darkPalette)
	if idle.HasText(i18n.Lookup(i18n.LocaleEN, i18n.KeyClosingTitle)) {
		t.Fatalf("a running app painted the closing layer; texts = %q", idle.Texts())
	}
	if got := idle.Image().RGBAAt(1, 1); got.R != page || got.G != 30 || got.B != 30 {
		t.Fatalf("the running app's page is covered: %v, want %v", got, pageRed)
	}
}
