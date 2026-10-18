//go:build darwin && (amd64 || arm64)

package darwin

import "github.com/egoist/mygo/internal/platform"

func (w *window) resetProductKeys() {
	if receiver, ok := w.h.(interface{ ProductWebKeyReset() }); ok {
		receiver.ProductWebKeyReset()
	}
}

// Called by our public NSWindow subclass before normal responder dispatch.
// Never calls private WKContentView selectors or injects a page bridge.
func (w *window) productKeyEvent(ev id) bool {
	// Embedded browser responders belong to the parent NSWindow now.
	for child := range browserSeats {
		if child.parent == w && child.browserHasFocus() {
			return child.productKeyEvent(ev)
		}
	}
	typ := sendInt(ev, "type")
	if typ != 10 && typ != 11 && typ != 12 {
		return false
	} // down/up/flagsChanged
	if w.closed || w.web == 0 || w.opts.BrowserProfile || w.opts.IsolatedPreview {
		return false
	}
	receiver, ok := w.h.(interface {
		ProductWebKeyEvent(uint32, bool, bool, bool, bool, bool, bool) bool
	})
	if !ok {
		return false
	}
	focus := send(w.browserNativeWindow(), "firstResponder")
	if focus == 0 || !respondsTo(focus, "isDescendantOf:") || (focus != w.web && !sendBool(focus, "isDescendantOf:", uintptr(w.web))) {
		w.resetProductKeys()
		return false
	}
	if typ == 12 || (respondsTo(focus, "hasMarkedText") && sendBool(focus, "hasMarkedText")) {
		w.resetProductKeys()
		return false
	}
	flags := uint64(send(ev, "modifierFlags"))
	ctrl, alt, shift, meta := flags&(1<<18) != 0, flags&(1<<19) != 0, flags&(1<<17) != 0, flags&(1<<20) != 0
	// Option alone is also AppKit's dead-key/text-composition prefix. Let the
	// renderer/input method own it even before marked text has appeared.
	if alt && !ctrl && !meta {
		w.resetProductKeys()
		return false
	}
	vk := platform.ProductMacVirtualKey(uint16(send(ev, "keyCode")))
	if vk == 0 {
		w.resetProductKeys()
		return false
	}
	return receiver.ProductWebKeyEvent(vk, ctrl, alt, shift, meta, sendBool(ev, "isARepeat"), typ == 10)
}
