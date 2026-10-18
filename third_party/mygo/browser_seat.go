package mygo

import "github.com/egoist/mygo/internal/platform"

// AttachBrowserSeat opts a hidden child URL window into live, same-window
// composition with its native-UI Parent where supported (currently Wayland).
// It preserves the page and its security/profile configuration. False leaves
// the ordinary child-window behavior unchanged. Call before showing the child.
func (w *Window) AttachBrowserSeat() bool {
	return get(w, func(n platform.Window) bool {
		if w.parent == nil || w.parent.native == nil {
			return false
		}
		seat, ok := n.(interface{ AttachBrowserSeat(platform.Window) bool })
		return ok && seat.AttachBrowserSeat(w.parent.native)
	})
}

// SetBrowserSeatBounds sets parent-content-relative logical bounds for an
// attached seat. False means this window is not an embedded seat.
func (w *Window) SetBrowserSeatBounds(r Rectangle) bool {
	return get(w, func(n platform.Window) bool {
		seat, ok := n.(interface{ SetBrowserSeatBounds(platform.Rect) bool })
		return ok && seat.SetBrowserSeatBounds(platform.Rect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height})
	})
}

// SetBrowserSeatOcclusions excludes seat-local rectangles from both live page
// drawing and pointer delivery. Nil restores the full seat. False means this
// window is not embedded; callers can retain their platform clipping path.
func (w *Window) SetBrowserSeatOcclusions(holes []Rectangle) bool {
	copy := make([]platform.Rect, len(holes))
	for i, r := range holes {
		copy[i] = platform.Rect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
	}
	return get(w, func(n platform.Window) bool {
		seat, ok := n.(interface{ SetBrowserSeatOcclusions([]platform.Rect) bool })
		return ok && seat.SetBrowserSeatOcclusions(copy)
	})
}
