//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestPrototypeMouseCDPExactParameters(t *testing.T) {
	for _, tc := range []struct {
		event        platform.BrowserMouseEvent
		kind, button string
	}{
		{platform.BrowserMouseEvent{Kind: platform.BrowserMouseMove, X: 10.5, Y: 20.25}, "mouseMoved", "none"},
		{platform.BrowserMouseEvent{Kind: platform.BrowserMousePress, Button: platform.BrowserMouseLeft, Buttons: 1, ClickCount: 1, Modifiers: 10}, "mousePressed", "left"},
		{platform.BrowserMouseEvent{Kind: platform.BrowserMouseRelease, Button: platform.BrowserMouseRight, ClickCount: 1}, "mouseReleased", "right"},
		{platform.BrowserMouseEvent{Kind: platform.BrowserMouseWheel, X: 50, Y: 60, DeltaY: 120, Modifiers: 2}, "mouseWheel", "none"},
	} {
		data, err := browserMouseCDPParams(tc.event)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err = json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 10 || got["type"] != tc.kind || got["button"] != tc.button || got["pointerType"] != "mouse" || got["x"] != tc.event.X || got["y"] != tc.event.Y || got["buttons"] != float64(tc.event.Buttons) || got["modifiers"] != float64(tc.event.Modifiers) || got["clickCount"] != float64(tc.event.ClickCount) || got["deltaX"] != tc.event.DeltaX || got["deltaY"] != tc.event.DeltaY {
			t.Fatalf("CDP=%s", data)
		}
	}
	if _, err := browserMouseCDPParams(platform.BrowserMouseEvent{Kind: platform.BrowserMouseCancel}); err == nil {
		t.Fatal("made-up mouseCancelled accepted")
	}
	var w window
	if w.BrowserMouseInputAvailable() {
		t.Fatal("uninitialized WebView advertised input")
	}
}
