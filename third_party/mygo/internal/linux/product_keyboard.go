//go:build linux && (amd64 || arm64)

package linux

import (
	"github.com/ebitengine/purego"
	"github.com/egoist/mygo/internal/platform"
)

var productWindowFocus func(ptr) ptr
var productWidgetAncestor func(ptr, ptr) bool
var productIMContext func(ptr) ptr
var productPreedit func(ptr, *ptr, *ptr, *uint32)
var productPeekEvent func() ptr
var cbProductKey, cbProductPreeditStart, cbProductPreeditEnd ptr

func loadProductKeyboardGTK(lib ptr) {
	bind(lib, &productWindowFocus, "gtk_window_get_focus")
	bind(lib, &productWidgetAncestor, "gtk_widget_is_ancestor")
	bind(libGDK, &productPeekEvent, "gdk_event_peek")
}
func loadProductKeyboardWebKit(lib ptr) {
	bind(lib, &productIMContext, "webkit_web_view_get_input_method_context")
	bind(lib, &productPreedit, "webkit_input_method_context_get_preedit")
}
func initProductKeyboardCallbacks() {
	cbProductKey = purego.NewCallback(func(widget, event, data ptr) bool {
		w := theBackend.window(data)
		if w == nil {
			return false
		}
		return w.productKeyEvent(event)
	})
	cbProductPreeditStart = purego.NewCallback(func(context, data ptr) {
		if w := theBackend.window(data); w != nil && context == w.productIM {
			w.productComposing = true
			w.resetProductKeys()
		}
	})
	cbProductPreeditEnd = purego.NewCallback(func(context, data ptr) {
		if w := theBackend.window(data); w != nil && context == w.productIM {
			w.productComposing = false
			w.resetProductKeys()
		}
	})
}
func (w *window) resetProductKeys() {
	w.productDown = nil
	if receiver, ok := w.h.(interface{ ProductWebKeyReset() }); ok {
		receiver.ProductWebKeyReset()
	}
}
func (w *window) productKeyEvent(event ptr) bool {
	if w.closed || w.web == 0 || w.opts.BrowserProfile || w.opts.IsolatedPreview {
		return false
	}
	receiver, ok := w.h.(interface {
		ProductWebKeyEvent(uint32, bool, bool, bool, bool, bool, bool) bool
	})
	if !ok {
		return false
	}
	if productWindowFocus == nil || productWidgetAncestor == nil || productIMContext == nil || productPreedit == nil {
		w.resetProductKeys()
		return false
	}
	top := w.win
	if w.seat != nil {
		top = w.seat.parent.win
	}
	focus := productWindowFocus(top)
	if focus == 0 || (focus != w.web && !productWidgetAncestor(focus, w.web)) {
		w.resetProductKeys()
		return false
	}
	im := productIMContext(w.web)
	if im != w.productIM {
		w.productIM = im
		w.productComposing = false
		w.resetProductKeys()
		if im != 0 {
			connect(im, "preedit-started", cbProductPreeditStart, ptr(w.id))
			connect(im, "preedit-finished", cbProductPreeditEnd, ptr(w.id))
		}
	}
	if im != 0 {
		var text ptr
		productPreedit(im, &text, nil, nil)
		composing := text != 0 && goStr(text) != ""
		if text != 0 {
			gFree(text)
		}
		if composing {
			w.resetProductKeys()
			return false
		}
	}
	state, key := field[uint32](event, 24), field[uint32](event, 28)
	// Leave alternate layout groups, AltGr/Level3/Level5, dead and compose keys
	// to the input method. This hook observes preedit; it never filters it twice.
	if w.productComposing || field[uint8](event, 50) != 0 || state&(1<<5|1<<7) != 0 || key == 0xff20 || (key >= 0xfe50 && key <= 0xfe93) || key == 0xfe03 || key == 0xfe11 {
		w.resetProductKeys()
		return false
	}
	code := field[uint16](event, 48)
	vk := platform.ProductGDKVirtualKey(code)
	if vk == 0 {
		w.resetProductKeys()
		return false
	}
	down := field[int32](event, 0) == 8
	// Some X11 servers synthesize release/press pairs for auto-repeat. A queued
	// press with identical time and hardware code is not a real key release.
	if !down && productPeekEvent != nil {
		next := productPeekEvent()
		if next != 0 {
			auto := field[int32](next, 0) == 8 && field[uint32](next, 20) == field[uint32](event, 20) && field[uint16](next, 48) == code
			gdkEventFree(next)
			if auto {
				return w.productDown[code]
			}
		}
	}
	_, repeat := w.productDown[code]
	consumed := receiver.ProductWebKeyEvent(vk, state&(1<<2) != 0, state&(1<<3) != 0, state&1 != 0, state&(1<<26|1<<28) != 0, repeat, down)
	if down {
		if w.productDown == nil {
			w.productDown = map[uint16]bool{}
		}
		w.productDown[code] = consumed
	} else {
		delete(w.productDown, code)
	}
	return consumed
}

// ResetProductKeyboard revokes consumption without making a still-held key
// look like a fresh press in the new registry. Real release removes its entry.
func (w *window) ResetProductKeyboard() {
	for code := range w.productDown {
		w.productDown[code] = false
	}
}
