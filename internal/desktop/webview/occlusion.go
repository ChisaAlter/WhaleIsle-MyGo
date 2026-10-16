package webview

import "github.com/egoist/mygo"

// SetOcclusions subtracts in-app overlay rectangles (parent-content DIPs) from
// the live native browser window. It never hides or recreates the page. A nil
// list restores the full native surface after redocking/closing the last float.
func (h *Host) SetOcclusions(rects []mygo.Rectangle) {
	if h == nil {
		return
	}
	h.mu.Lock()
	win, parent := h.win, h.parent
	h.mu.Unlock()
	if win == nil || parent == nil || win.IsDestroyed() || parent.IsDestroyed() {
		return
	}
	bounds, origin := win.Bounds(), parent.ContentBounds()
	local := make([]mygo.Rectangle, 0, len(rects))
	for _, r := range rects {
		r.X += origin.X - bounds.X
		r.Y += origin.Y - bounds.Y
		if r.X+r.Width <= 0 || r.Y+r.Height <= 0 || r.X >= bounds.Width || r.Y >= bounds.Height {
			continue
		}
		local = append(local, r)
	}
	setNativeOcclusions(win, bounds.Width, bounds.Height, local)
}
