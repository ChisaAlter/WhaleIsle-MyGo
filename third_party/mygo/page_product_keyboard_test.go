package mygo

import "testing"

func TestProductKeyboardNativePriority(t *testing.T) {
	calls := 0
	w := &Window{productKeyboard: &productKeyboardState{revision: "r", active: true, bindings: []ProductKeyBinding{{Code: "KeyP", Mods: 5}}, dispatch: func(r, c string, m uint8, second string) {
		if r != "r" || c != "KeyP" || m != 5 || second != "" {
			t.Fatal("gesture mutated")
		}
		calls++
	}}}
	h := &windowHandler{w: w}
	if !h.ProductWebKey(80, true, false, true, false, false) || calls != 1 {
		t.Fatal("accepted key not consumed")
	}
	if !h.ProductWebKey(80, true, false, true, false, true) || calls != 1 {
		t.Fatal("repeat executed")
	}
	if h.ProductWebKey(70, true, false, true, false, false) {
		t.Fatal("unregistered key consumed")
	}
	w.productKeyboard.active = false
	if h.ProductWebKey(80, true, false, true, false, false) {
		t.Fatal("guarded key consumed")
	}
}
func TestProductKeyboardHeldPairs(t *testing.T) {
	calls := 0
	w := &Window{productKeyboard: &productKeyboardState{revision: "r", active: true, bindings: []ProductKeyBinding{{Code: "KeyA", SecondCode: "KeyB"}}, dispatch: func(r, c string, m uint8, second string) {
		if c != "KeyA" || second != "KeyB" {
			t.Fatal("pair mutated")
		}
		calls++
	}}}
	h := &windowHandler{w: w}
	key := func(vk uint32, repeat, down bool) bool {
		return h.ProductWebKeyEvent(vk, false, false, false, false, repeat, down)
	}
	if key(66, false, true) || !key(65, false, true) || calls != 1 {
		t.Fatal("unordered held pair missing")
	}
	if !key(65, true, true) || !key(66, true, true) || calls != 1 {
		t.Fatal("repeat dispatch")
	}
	if key(66, false, false) || !key(65, false, false) {
		t.Fatal("release ownership differs from intercepted press")
	}
	key(65, false, true)
	key(65, false, false)
	if key(66, false, true) || calls != 1 {
		t.Fatal("sequence mistaken for held pair")
	}
	h.ProductWebKeyReset()
	key(65, false, true)
	key(67, false, true)
	if key(66, false, true) {
		t.Fatal("three-key chord admitted")
	}
	h.ProductWebKeyReset()
	key(65, false, true)
	h.ProductWebKeyReset()
	if key(66, false, true) {
		t.Fatal("reset retained held state")
	}
	h.ProductWebKeyReset()
	key(65, false, true)
	if h.ProductWebKeyEvent(66, true, false, false, false, false, true) {
		t.Fatal("modifier transition retained chord")
	}
}
func TestProductKeyPhysicalCodes(t *testing.T) {
	for _, code := range []string{"KeyA", "Digit9", "F24", "ArrowLeft", "Escape", "Space"} {
		if productCodeVK(code) == 0 {
			t.Fatal(code)
		}
	}
	if productCodeVK("arbitrary") != 0 {
		t.Fatal("unknown code accepted")
	}
}
