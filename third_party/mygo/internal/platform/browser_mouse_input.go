package platform

import (
	"fmt"
	"math"
)

// BrowserMouseKind is one bounded host-originated mouse operation. Cancel is a
// best-effort release sequence assembled by the public layer, not a CDP event.
type BrowserMouseKind uint8

const (
	BrowserMouseMove BrowserMouseKind = iota
	BrowserMousePress
	BrowserMouseRelease
	BrowserMouseWheel
	BrowserMouseCancel
)

type BrowserMouseButton uint8

const (
	BrowserMouseNone BrowserMouseButton = iota
	BrowserMouseLeft
	BrowserMouseMiddle
	BrowserMouseRight
	BrowserMouseBack
	BrowserMouseForward
)

// BrowserMouseEvent uses CSS pixels relative to the MAIN frame viewport.
// Buttons is the CDP/DOM mask: Left=1, Right=2, Middle=4, Back=8, Forward=16.
// Modifiers is the CDP mask: Alt=1, Ctrl=2, Meta=4, Shift=8. Button identifies
// the changed button on Press/Release; Buttons is the state AFTER that change.
// ClickCount is 1 for an ordinary Press/Release; Move/Wheel use zero.
// DeltaX/DeltaY are CSS pixels, positive right/down, only for Wheel.
type BrowserMouseEvent struct {
	Kind               BrowserMouseKind
	X, Y               float64
	Button             BrowserMouseButton
	Buttons, Modifiers uint8
	ClickCount         int
	DeltaX, DeltaY     float64
}

func BrowserMouseButtonMask(button BrowserMouseButton) uint8 {
	switch button {
	case BrowserMouseLeft:
		return 1
	case BrowserMouseRight:
		return 2
	case BrowserMouseMiddle:
		return 4
	case BrowserMouseBack:
		return 8
	case BrowserMouseForward:
		return 16
	}
	return 0
}

func ValidateBrowserMouseEvent(e BrowserMouseEvent) error {
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	if e.Kind > BrowserMouseCancel || e.Button > BrowserMouseForward || e.Buttons & ^uint8(31) != 0 || e.Modifiers & ^uint8(15) != 0 || e.ClickCount < 0 || e.ClickCount > 3 {
		return fmt.Errorf("mygo: invalid mouse event kind, button, mask or click count")
	}
	if !finite(e.X) || !finite(e.Y) || !finite(e.DeltaX) || !finite(e.DeltaY) {
		return fmt.Errorf("mygo: mouse coordinates and deltas must be finite")
	}
	if e.Kind == BrowserMouseCancel {
		if e.Button != BrowserMouseNone || e.Buttons != 0 || e.Modifiers != 0 || e.ClickCount != 0 || e.DeltaX != 0 || e.DeltaY != 0 {
			return fmt.Errorf("mygo: cancel accepts no button, delta or modifier payload")
		}
		return nil
	}
	if e.Kind != BrowserMouseWheel && (e.DeltaX != 0 || e.DeltaY != 0) {
		return fmt.Errorf("mygo: mouse deltas require a wheel event")
	}
	if e.Kind == BrowserMousePress || e.Kind == BrowserMouseRelease {
		mask := BrowserMouseButtonMask(e.Button)
		if mask == 0 || e.Kind == BrowserMousePress && e.Buttons&mask == 0 || e.Kind == BrowserMouseRelease && e.Buttons&mask != 0 {
			return fmt.Errorf("mygo: changed mouse button disagrees with buttons mask")
		}
	} else if e.Button != BrowserMouseNone || e.ClickCount != 0 {
		return fmt.Errorf("mygo: mouse move/wheel requires no changed button or click count")
	}
	return nil
}

// BrowserMouseInputWindow is optional. A missing provider is unsupported; no
// new mandatory Window method or platform-specific no-op support is added.
// Calls and completion callbacks run on the UI thread. Ready must not queue
// input for a future WebView/document. The public layer handles Cancel.
type BrowserMouseInputWindow interface {
	BrowserMouseInputAvailable() bool
	DispatchBrowserMouseInput(BrowserMouseEvent, func(error))
}

func BrowserMouseInputProvider(w any) BrowserMouseInputWindow {
	n, ok := w.(BrowserMouseInputWindow)
	if !ok || !n.BrowserMouseInputAvailable() {
		return nil
	}
	return n
}
