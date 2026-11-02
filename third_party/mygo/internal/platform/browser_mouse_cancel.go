package platform

// BrowserMouseCancelEvents is a best-effort release/leave sequence, not an
// invented mouseCancelled CDP kind. A renderer with pointer capture can still
// deliver releases to the captured element; product gesture gates may disarm it.
func BrowserMouseCancelEvents(held uint8) []BrowserMouseEvent {
	events := []BrowserMouseEvent{{Kind: BrowserMouseMove, X: -1, Y: -1, Buttons: held}}
	for _, button := range []BrowserMouseButton{BrowserMouseLeft, BrowserMouseRight, BrowserMouseMiddle, BrowserMouseBack, BrowserMouseForward} {
		mask := BrowserMouseButtonMask(button)
		if held&mask == 0 {
			continue
		}
		held &^= mask
		events = append(events, BrowserMouseEvent{Kind: BrowserMouseRelease, X: -1, Y: -1, Button: button, Buttons: held})
	}
	return append(events, BrowserMouseEvent{Kind: BrowserMouseMove, X: -1, Y: -1})
}
