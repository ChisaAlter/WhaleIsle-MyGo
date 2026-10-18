package darwin

import "github.com/egoist/mygo/internal/platform"

func browserSeatContains(width, height, x, y float64, holes []platform.Rect) bool {
	if x < 0 || y < 0 || x >= width || y >= height {
		return false
	}
	for _, r := range holes {
		if r.Width > 0 && r.Height > 0 && x >= float64(r.X) && y >= float64(r.Y) && x < float64(r.X+r.Width) && y < float64(r.Y+r.Height) {
			return false
		}
	}
	return true
}

// Rectangle subtraction rather than even-odd holes: overlapping floats must
// remain excluded instead of punching an accidentally visible overlap back in.
func browserSeatVisible(bounds platform.Rect, holes []platform.Rect) []platform.Rect {
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return nil
	}
	pieces := []platform.Rect{bounds}
	for _, h := range holes {
		if h.Width <= 0 || h.Height <= 0 {
			continue
		}
		next := make([]platform.Rect, 0, len(pieces))
		for _, r := range pieces {
			x0, y0 := max(r.X, h.X), max(r.Y, h.Y)
			x1, y1 := min(r.X+r.Width, h.X+h.Width), min(r.Y+r.Height, h.Y+h.Height)
			if x0 >= x1 || y0 >= y1 {
				next = append(next, r)
				continue
			}
			for _, p := range []platform.Rect{
				{X: r.X, Y: r.Y, Width: r.Width, Height: y0 - r.Y},
				{X: r.X, Y: y1, Width: r.Width, Height: r.Y + r.Height - y1},
				{X: r.X, Y: y0, Width: x0 - r.X, Height: y1 - y0},
				{X: x1, Y: y0, Width: r.X + r.Width - x1, Height: y1 - y0},
			} {
				if p.Width > 0 && p.Height > 0 {
					next = append(next, p)
				}
			}
		}
		pieces = next
	}
	return pieces
}
