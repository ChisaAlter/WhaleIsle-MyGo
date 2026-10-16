//go:build linux && (amd64 || arm64)

package linux

import (
	"github.com/ebitengine/purego"
	"github.com/egoist/mygo/internal/platform"
	"sync"
	"unsafe"
)

// browserSeat retains the existing WebKit widget/context. The original hidden
// GtkWindow remains its logical owner; only the widget joins the parent tree.
type browserSeat struct {
	parent *window
	bounds platform.Rect
	holes  []platform.Rect
	region ptr
}

var seatGTK struct {
	sync.Once
	parent, parentWindow                             func(ptr) ptr
	queueResize                                      func(ptr)
	regionCreate                                     func(*gdkRectangle) ptr
	regionStatus                                     func(ptr) int32
	regionSubtract                                   func(ptr, *gdkRectangle) int32
	regionDestroy                                    func(ptr)
	cairoRegion                                      func(ptr, ptr)
	cairoClip                                        func(ptr)
	inputShape                                       func(ptr, ptr, int32, int32)
	position, draw, allocate, focusIn, focusOut, key ptr
}

func loadBrowserSeat() {
	seatGTK.Do(func() {
		mustBind(libGTK, &seatGTK.parent, "gtk_widget_get_parent")
		mustBind(libGTK, &seatGTK.parentWindow, "gtk_widget_get_parent_window")
		mustBind(libGTK, &seatGTK.queueResize, "gtk_widget_queue_resize")
		mustBind(libCairo, &seatGTK.regionCreate, "cairo_region_create_rectangle")
		mustBind(libCairo, &seatGTK.regionStatus, "cairo_region_status")
		mustBind(libCairo, &seatGTK.regionSubtract, "cairo_region_subtract_rectangle")
		mustBind(libCairo, &seatGTK.regionDestroy, "cairo_region_destroy")
		mustBind(libGDK, &seatGTK.cairoRegion, "gdk_cairo_region")
		mustBind(libCairo, &seatGTK.cairoClip, "cairo_clip")
		mustBind(libGDK, &seatGTK.inputShape, "gdk_window_input_shape_combine_region")
		seatGTK.position = purego.NewCallback(func(overlay, child, allocation, data ptr) bool {
			p := theBackend.window(data)
			if p == nil {
				return false
			}
			for _, w := range p.seats {
				if w.web == child && w.seat != nil {
					r := w.seat.bounds
					*(*gdkRectangle)(unsafe.Pointer(allocation)) = gdkRectangle{X: int32(r.X), Y: int32(r.Y), Width: int32(r.Width), Height: int32(r.Height)}
					return true
				}
			}
			return false
		})
		seatGTK.draw = purego.NewCallback(func(widget, cr, data ptr) bool {
			if w := theBackend.window(data); w != nil && w.seat != nil && w.seat.region != 0 {
				// GTK propagates this clip into WebKit's default draw handler. No
				// snapshot or page replacement; animation and scrolling remain live.
				seatGTK.cairoRegion(cr, w.seat.region)
				seatGTK.cairoClip(cr)
			}
			return false
		})
		seatGTK.allocate = purego.NewCallback(func(widget, allocation, data ptr) {
			if w := theBackend.window(data); w != nil {
				w.applySeatInput()
			}
		})
		seatGTK.focusIn = purego.NewCallback(func(widget, event, data ptr) bool {
			if w := theBackend.window(data); w != nil {
				w.h.Focused()
			}
			return false
		})
		seatGTK.focusOut = purego.NewCallback(func(widget, event, data ptr) bool {
			if w := theBackend.window(data); w != nil {
				w.resetProductKeys()
				w.h.Blurred()
			}
			return false
		})
		seatGTK.key = purego.NewCallback(func(widget, event, data ptr) bool {
			p := theBackend.window(data)
			if p == nil {
				return false
			}
			for _, w := range p.seats {
				if w.seatFocused() {
					return w.productKeyEvent(event)
				}
			}
			return false
		})
	})
}

func (w *window) AttachBrowserSeat(parent platform.Window) bool {
	p, ok := parent.(*window)
	if !ok || p == nil || p.closed || p.surface == nil || w.closed || w.web == 0 || w.b.onX11 || w.opts.Parent != parent || w.opts.Modal || w.controls != nil || gtkWidgetGetVisible(w.win) {
		return false
	}
	if w.seat != nil {
		return w.seat.parent == p
	}
	loadBrowserSeat()
	if p.seatOverlay == 0 {
		// Reparent only on explicit opt-in. GTK's existing unrealize/realize
		// handlers release/recreate the surface GL resources as usual.
		area := p.surface.area
		old := seatGTK.parent(area)
		gObjectRef(area)
		gtkContainerRemove(old, area)
		p.seatOverlay = gtkOverlayNew()
		gtkContainerAdd(p.seatOverlay, area)
		gObjectUnref(area)
		if p.controls != nil {
			gtkContainerAdd(old, p.seatOverlay)
		} else {
			gtkBoxPackStart(p.box, p.seatOverlay, true, true, 0)
		}
		connect(p.seatOverlay, "get-child-position", seatGTK.position, ptr(p.id))
		connect(p.win, "key-press-event", seatGTK.key, ptr(p.id))
		connect(p.win, "key-release-event", seatGTK.key, ptr(p.id))
		gtkWidgetShow(p.seatOverlay)
	}
	w.seat = &browserSeat{parent: p, bounds: platform.Rect{Width: 1, Height: 1}}
	p.seats = append(p.seats, w)
	gtkWidgetHide(w.web)
	gObjectRef(w.web)
	gtkContainerRemove(seatGTK.parent(w.web), w.web)
	gtkOverlayAddOverlay(p.seatOverlay, w.web)
	// Keep this reference until destroySeat: parent destruction may destroy the
	// widget before the logical child owner receives its close notification.
	connect(w.web, "draw", seatGTK.draw, ptr(w.id))
	connect(w.web, "size-allocate", seatGTK.allocate, ptr(w.id))
	connect(w.web, "realize", cbSeatRealize(), ptr(w.id))
	connect(w.web, "focus-in-event", seatGTK.focusIn, ptr(w.id))
	connect(w.web, "focus-out-event", seatGTK.focusOut, ptr(w.id))
	w.rebuildSeatRegion()
	return true
}

// The size-allocate callback also runs after realization; no per-seat callback
// allocation is permitted. A separate signature is registered once below.
var seatRealizeOnce sync.Once
var seatRealize ptr

func cbSeatRealize() ptr {
	seatRealizeOnce.Do(func() {
		seatRealize = purego.NewCallback(func(widget, data ptr) {
			if w := theBackend.window(data); w != nil {
				w.applySeatInput()
			}
		})
	})
	return seatRealize
}

func (w *window) SetBrowserSeatBounds(r platform.Rect) bool {
	if w.seat == nil {
		return false
	}
	r.Width, r.Height = max(1, min(r.Width, 1<<30)), max(1, min(r.Height, 1<<30))
	r.X, r.Y = max(0, min(r.X, 1<<30)), max(0, min(r.Y, 1<<30))
	w.seat.bounds = r
	w.rebuildSeatRegion()
	seatGTK.queueResize(w.seat.parent.seatOverlay)
	return true
}
func (w *window) SetBrowserSeatOcclusions(holes []platform.Rect) bool {
	if w.seat == nil {
		return false
	}
	w.seat.holes = append([]platform.Rect(nil), holes...)
	w.rebuildSeatRegion()
	return true
}
func (w *window) rebuildSeatRegion() {
	s := w.seat
	r := gdkRectangle{Width: int32(s.bounds.Width), Height: int32(s.bounds.Height)}
	region := seatGTK.regionCreate(&r)
	if region == 0 {
		return
	}
	if seatGTK.regionStatus(region) != 0 {
		seatGTK.regionDestroy(region)
		return
	}
	for _, h := range s.holes {
		cut, ok := browserSeatClipRect(s.bounds.Width, s.bounds.Height, h)
		if !ok {
			continue
		}
		if seatGTK.regionSubtract(region, &cut) != 0 {
			seatGTK.regionDestroy(region)
			return
		}
	}
	previous := s.region
	s.region = region
	w.applySeatInput()
	gtkWidgetQueueDraw(s.parent.seatOverlay)
	if previous != 0 {
		seatGTK.regionDestroy(previous)
	}
}
func (w *window) applySeatInput() {
	if w.seat == nil || w.seat.region == 0 {
		return
	}
	// GtkOverlay gives each overlay child a distinct GDK child window.
	// Shape that wrapper, not merely WebKit's descendant: otherwise the
	// wrapper itself would still intercept pointer events over the holes.
	if wrapper := seatGTK.parentWindow(w.web); wrapper != 0 {
		seatGTK.inputShape(wrapper, w.seat.region, 0, 0)
	}
}
func (w *window) seatFocused() bool {
	if w.seat == nil || productWindowFocus == nil || productWidgetAncestor == nil {
		return false
	}
	focus := productWindowFocus(w.seat.parent.win)
	return focus != 0 && (focus == w.web || productWidgetAncestor(focus, w.web))
}
func (w *window) destroySeat() {
	if w.seat == nil {
		return
	}
	s := w.seat
	w.seat = nil
	p := s.parent
	for i, child := range p.seats {
		if child == w {
			p.seats = append(p.seats[:i], p.seats[i+1:]...)
			break
		}
	}
	if s.region != 0 {
		seatGTK.regionDestroy(s.region)
	}
	// The widget no longer belongs to the hidden owner GtkWindow.
	gtkWidgetDestroy(w.web)
	gObjectUnref(w.web)
}

// browserSeatClipRect uses bounded endpoints before narrowing to GTK ints;
// hostile/extreme rectangles cannot overflow endpoint arithmetic.
func browserSeatClipRect(width, height int, h platform.Rect) (gdkRectangle, bool) {
	if width <= 0 || height <= 0 || h.Width <= 0 || h.Height <= 0 {
		return gdkRectangle{}, false
	}
	x, y := max(0, h.X), max(0, h.Y)
	right, bottom := width, height
	if h.X <= width-h.Width {
		right = h.X + h.Width
	}
	if h.Y <= height-h.Height {
		bottom = h.Y + h.Height
	}
	if x >= right || y >= bottom {
		return gdkRectangle{}, false
	}
	return gdkRectangle{X: int32(x), Y: int32(y), Width: int32(right - x), Height: int32(bottom - y)}, true
}
