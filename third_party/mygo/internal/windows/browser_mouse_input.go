//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/json"
	"errors"

	"github.com/egoist/mygo/internal/platform"
)

func (w *window) BrowserMouseInputAvailable() bool {
	return !w.closed && !w.destroying && w.surface == nil && w.ready && w.webview != 0
}

// DispatchBrowserMouseInput uses the existing WebView2 CDP COM adapter. It
// opens no DevTools UI and creates no callbacks per event (withHandler reuses
// the static signature vtable). It never queues input for a not-yet-ready page.
func (w *window) DispatchBrowserMouseInput(e platform.BrowserMouseEvent, done func(error)) {
	if !w.BrowserMouseInputAvailable() {
		done(errors.New("mygo: WebView2 is not ready for mouse input"))
		return
	}
	params, err := browserMouseCDPParams(e)
	if err != nil {
		done(err)
		return
	}
	w.devtools("Input.dispatchMouseEvent", string(params), func(_ string, err error) { done(err) })
}

func browserMouseCDPParams(e platform.BrowserMouseEvent) ([]byte, error) {
	if err := platform.ValidateBrowserMouseEvent(e); err != nil {
		return nil, err
	}
	var kind string
	switch e.Kind {
	case platform.BrowserMouseMove:
		kind = "mouseMoved"
	case platform.BrowserMousePress:
		kind = "mousePressed"
	case platform.BrowserMouseRelease:
		kind = "mouseReleased"
	case platform.BrowserMouseWheel:
		kind = "mouseWheel"
	default:
		return nil, errors.New("mygo: cancel must be translated into typed release events")
	}
	button := "none"
	switch e.Button {
	case platform.BrowserMouseLeft:
		button = "left"
	case platform.BrowserMouseMiddle:
		button = "middle"
	case platform.BrowserMouseRight:
		button = "right"
	case platform.BrowserMouseBack:
		button = "back"
	case platform.BrowserMouseForward:
		button = "forward"
	}
	// Keep zero-valued buttons, wheel axes and clickCount explicit. This avoids
	// accidentally retaining a previous press or dropping the zero wheel axis.
	return json.Marshal(struct {
		Type        string  `json:"type"`
		X           float64 `json:"x"`
		Y           float64 `json:"y"`
		Button      string  `json:"button"`
		Buttons     uint8   `json:"buttons"`
		Modifiers   uint8   `json:"modifiers"`
		ClickCount  int     `json:"clickCount"`
		DeltaX      float64 `json:"deltaX"`
		DeltaY      float64 `json:"deltaY"`
		PointerType string  `json:"pointerType"`
	}{kind, e.X, e.Y, button, e.Buttons, e.Modifiers, e.ClickCount, e.DeltaX, e.DeltaY, "mouse"})
}
