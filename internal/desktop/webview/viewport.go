package webview

import "github.com/egoist/mygo"

// SetViewport selects a parent-content-relative seat. Nil restores the original
// full panel behavior. Parent moves/resizes keep using this seat, not full width.
func (h *Host) SetViewport(rect *mygo.Rectangle) {
	if h == nil {
		return
	}
	h.mu.Lock()
	if rect == nil {
		h.viewport = nil
	} else {
		r := *rect
		h.viewport = &r
	}
	h.mu.Unlock()
	h.SyncBounds()
}
func viewportBounds(parent mygo.Rectangle, inset int, seat *mygo.Rectangle) mygo.Rectangle {
	if seat == nil {
		return mygo.Rectangle{X: parent.X, Y: parent.Y + inset, Width: parent.Width, Height: max(1, parent.Height-inset)}
	}
	x := max(0, min(seat.X, parent.Width))
	y := max(0, min(seat.Y, parent.Height))
	return mygo.Rectangle{X: parent.X + x, Y: parent.Y + y, Width: max(1, min(seat.Width, parent.Width-x)), Height: max(1, min(seat.Height, parent.Height-y))}
}
