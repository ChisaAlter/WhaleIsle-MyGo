package ui

import "github.com/egoist/mygo/internal/platform"

func (h *windowHost) zoomFactor() float64 {
	if h.zoom <= 0 {
		return 1
	}
	return h.zoom
}

// ZoomFactor is the native content scale, independent of monitor DPI and fonts.
// The API is UI-thread-owned, like the Context it belongs to.
func (c *Context) ZoomFactor() float64 {
	if rt := c.runtime(); rt != nil {
		if h, ok := rt.host.(*windowHost); ok {
			return h.zoomFactor()
		}
	}
	return 1
}

// SetZoomFactor scales the entire native viewport, including images and layouts.
// Geometry passed to native IME, accessibility, menus and drag remains in DIPs.
func (c *Context) SetZoomFactor(factor float64) {
	if factor < 0.5 {
		factor = 0.5
	}
	if factor > 2 {
		factor = 2
	}
	rt := c.runtime()
	if rt == nil {
		return
	}
	h, ok := rt.host.(*windowHost)
	if !ok {
		return
	}
	if h.zoomFactor() == factor {
		return
	}
	old := h.zoomFactor()
	h.zoom = factor
	rt.pointerX *= float32(old / factor)
	rt.pointerY *= float32(old / factor)
	rt.redraw = false
	rt.requestFrame()
}

func zoomRect(r platform.RectF, z float64) platform.RectF {
	return platform.RectF{X: r.X * z, Y: r.Y * z, W: r.W * z, H: r.H * z}
}

type zoomTextClient struct {
	platform.TextInputClient
	zoom float64
}

func (c zoomTextClient) BoundsForRange(r platform.TextRange) (platform.RectF, platform.TextRange, bool) {
	b, actual, ok := c.TextInputClient.BoundsForRange(r)
	return zoomRect(b, c.zoom), actual, ok
}
func (c zoomTextClient) IndexForPoint(x, y float64) (int, bool) {
	return c.TextInputClient.IndexForPoint(x/c.zoom, y/c.zoom)
}
