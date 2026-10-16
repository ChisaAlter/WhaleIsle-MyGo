//go:build linux && (amd64 || arm64)

package linux

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
	"unsafe"
)

type productKeyFixture struct {
	platform.WindowHandler
	calls, resets int
	vk            uint32
	repeat, down  bool
	consume       bool
}

func (f *productKeyFixture) ProductWebKeyReset() { f.resets++ }
func (f *productKeyFixture) ProductWebKeyEvent(vk uint32, ctrl, alt, shift, meta, repeat, down bool) bool {
	f.calls++
	f.vk = vk
	f.repeat = repeat
	f.down = down
	return f.consume
}

// Static storage models a native event pointer without movable Go-stack memory.
var productEventFixture [64]byte

func TestProductNativeKeyRouting(t *testing.T) {
	oldFocus, oldAncestor, oldIM, oldPreedit, oldPeek := productWindowFocus, productWidgetAncestor, productIMContext, productPreedit, productPeekEvent
	t.Cleanup(func() {
		productWindowFocus, productWidgetAncestor, productIMContext, productPreedit, productPeekEvent = oldFocus, oldAncestor, oldIM, oldPreedit, oldPeek
	})
	productWindowFocus = func(ptr) ptr { return 10 }
	productWidgetAncestor = func(ptr, ptr) bool { return false }
	productIMContext = func(ptr) ptr { return 0 }
	productPreedit = func(ptr, *ptr, *ptr, *uint32) {}
	productPeekEvent = nil
	h := &productKeyFixture{consume: true}
	w := &window{h: h, web: 10, opts: &platform.WindowOptions{}}
	productEventFixture = [64]byte{}
	bytes := &productEventFixture
	event := ptr(unsafe.Pointer(&bytes[0]))
	*(*int32)(unsafe.Pointer(&bytes[0])) = 8
	*(*uint16)(unsafe.Pointer(&bytes[48])) = 38
	if !w.productKeyEvent(event) || h.vk != 'A' || h.repeat || !h.down {
		t.Fatal("first physical down not routed", h)
	}
	if !w.productKeyEvent(event) || !h.repeat {
		t.Fatal("repeat not detected")
	}
	*(*int32)(unsafe.Pointer(&bytes[0])) = 9
	if !w.productKeyEvent(event) || h.down {
		t.Fatal("release not routed")
	}
	*(*int32)(unsafe.Pointer(&bytes[0])) = 8
	if !w.productKeyEvent(event) || h.repeat {
		t.Fatal("fresh press marked repeat")
	}
	w.ResetProductKeyboard()
	if _, held := w.productDown[38]; !held || w.productDown[38] {
		t.Fatal("snapshot must retain held presence but revoke consumption")
	}
	previous := h.calls
	bytes[50] = 1
	if w.productKeyEvent(event) || h.calls != previous || len(w.productDown) != 0 {
		t.Fatal("alternate group intercepted")
	}
	bytes[50] = 0
	w.productComposing = true
	if w.productKeyEvent(event) || h.calls != previous {
		t.Fatal("IME intercepted")
	}
	w.productComposing = false
	productWindowFocus = func(ptr) ptr { return 20 }
	if w.productKeyEvent(event) || h.calls != previous {
		t.Fatal("other focused widget intercepted")
	}
	productWindowFocus = func(ptr) ptr { return 10 }
	productPreedit = nil
	if w.productKeyEvent(event) || h.calls != previous {
		t.Fatal("missing IME observation API accepted")
	}
	w.opts.BrowserProfile = true
	if w.productKeyEvent(event) || h.calls != previous {
		t.Fatal("ordinary Browser intercepted")
	}
}
