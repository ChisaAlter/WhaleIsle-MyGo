package frame

import (
	"sync"
	"weak"

	"github.com/egoist/mygo"
)

type displayWindowBinding struct {
	owner weak.Pointer[mygo.Window]
}

type displayWindowSource struct {
	mu      sync.Mutex
	binding *displayWindowBinding
}

var frameDisplayWindow displayWindowSource

// SetDisplayWindow binds the main frame's actual window, never a focused window
// or an arbitrary primary display. Call it after creating that main window; nil
// clears the binding. The source does not retain the window, and an old window's
// close callback cannot clear a newer binding. Native screen queries remain on
// the UI thread in the platform reader, not in this lifecycle hook.
func SetDisplayWindow(window *mygo.Window) {
	frameDisplayWindow.bind(window)
}

func (s *displayWindowSource) bind(window *mygo.Window) {
	s.bindOnClosed(window, func(owner *mygo.Window, closed func()) {
		// Do not retain OnClosed's unsubscribe closure here: it captures a
		// listener field inside Window and would keep the whole owner alive.
		// The window itself owns this callback, whose binding only holds weak.
		owner.OnClosed(closed)
	})
}

func (s *displayWindowSource) bindOnClosed(window *mygo.Window, onClosed func(*mygo.Window, func())) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.binding
	if old != nil && window != nil && old.owner.Value() == window && !window.IsDestroyed() {
		return
	}
	s.binding = nil
	if window != nil && !window.IsDestroyed() {
		next := &displayWindowBinding{owner: weak.Make(window)}
		onClosed(window, func() { s.clear(next) })
		// The close event may have fired between the first check and listener
		// registration. A callback waiting on mu will clear next after unlock.
		if !window.IsDestroyed() {
			s.binding = next
		}
	}
}

func (s *displayWindowSource) clear(expected *displayWindowBinding) {
	s.mu.Lock()
	if s.binding != expected {
		s.mu.Unlock()
		return
	}
	s.binding = nil
	s.mu.Unlock()
}

func (s *displayWindowSource) current() (*displayWindowBinding, *mygo.Window) {
	s.mu.Lock()
	binding := s.binding
	s.mu.Unlock()
	if binding == nil {
		return nil, nil
	}
	window := binding.owner.Value()
	if window == nil || window.IsDestroyed() {
		s.clear(binding)
		return nil, nil
	}
	return binding, window
}

// landscape accepts an on-UI-thread snapshot reader. Its pure seam lets headless
// tests exercise ownership, stale close callbacks and changing displays without
// initializing AppKit or making any window.
func (s *displayWindowSource) landscape(read func(*mygo.Window) (mygo.Rectangle, []mygo.Display, bool)) (landscape, known bool) {
	binding, window := s.current()
	if window == nil {
		return false, false
	}
	bounds, displays, ok := read(window)
	s.mu.Lock()
	current := s.binding == binding
	s.mu.Unlock()
	if !ok || !current || window.IsDestroyed() {
		return false, false
	}
	return displayWorkAreaLandscape(bounds, displays)
}

// displayWorkAreaLandscape selects only a uniquely best positive overlap with
// the actual window. Screen.DisplayMatching also returns the primary display
// when every overlap is zero, which would invent screen data for an offscreen
// window. Ambiguous, missing or square work areas keep the viewport fallback.
func displayWorkAreaLandscape(window mygo.Rectangle, displays []mygo.Display) (landscape, known bool) {
	if window.Width <= 0 || window.Height <= 0 {
		return false, false
	}
	best, bestArea, tied := -1, float64(0), false
	for i, display := range displays {
		bounds := display.Bounds
		if bounds.Width <= 0 || bounds.Height <= 0 {
			continue
		}
		// Convert before adding: virtual desktop coordinates may be negative,
		// and malformed integer bounds must not overflow into a false overlap.
		w := min(float64(bounds.X)+float64(bounds.Width), float64(window.X)+float64(window.Width)) - max(float64(bounds.X), float64(window.X))
		h := min(float64(bounds.Y)+float64(bounds.Height), float64(window.Y)+float64(window.Height)) - max(float64(bounds.Y), float64(window.Y))
		if w <= 0 || h <= 0 {
			continue
		}
		area := w * h
		if area > bestArea {
			best, bestArea, tied = i, area, false
		} else if area == bestArea {
			tied = true
		}
	}
	if best < 0 || tied {
		return false, false
	}
	work := displays[best].WorkArea
	if work.Width <= 0 || work.Height <= 0 || work.Width == work.Height {
		return false, false
	}
	return work.Width > work.Height, true
}
