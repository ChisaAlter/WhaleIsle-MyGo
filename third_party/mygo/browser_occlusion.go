package mygo

import "github.com/egoist/mygo/internal/platform"

// SetBrowserOcclusions embeds a supported browser child into its native parent
// and excludes these child-local DIP rectangles from rendering and input. It
// preserves the live page. A nil slice restores the complete browser seat.
// Returns false on backends without this optional embedding capability.
func (w *Window) SetBrowserOcclusions(rects []Rectangle) bool {
	holes := make([]platform.Rect, len(rects))
	for i, r := range rects {
		holes[i] = platform.Rect{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height}
	}
	return get(w, func(n platform.Window) bool {
		if seat, ok := n.(interface{ SetBrowserOcclusions([]platform.Rect) bool }); ok {
			return seat.SetBrowserOcclusions(holes)
		}
		return false
	})
}
