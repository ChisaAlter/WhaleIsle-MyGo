package mygo

import "testing"

func TestProductKeyboardNativePriority(t *testing.T) {
	calls := 0
	w := &Window{productKeyboard: &productKeyboardState{revision: "r", active: true, bindings: map[webShortcutKey]string{{80, 5}: "KeyP"}, dispatch: func(r, c string, m uint8) {
		if r != "r" || c != "KeyP" || m != 5 {
			t.Fatal("gesture mutated")
		}
		calls++
	}}}
	h := &windowHandler{w: w}
	if !h.ProductWebKey(80, true, false, true, false, false) || calls != 1 {
		t.Fatal("accepted registry key not consumed")
	}
	if !h.ProductWebKey(80, true, false, true, false, true) || calls != 1 {
		t.Fatal("repeat executed action")
	}
	if h.ProductWebKey(70, true, false, true, false, false) {
		t.Fatal("unregistered key consumed")
	}
	w.productKeyboard.active = false
	if h.ProductWebKey(80, true, false, true, false, false) {
		t.Fatal("local editor/recording/composition key consumed")
	}
}
