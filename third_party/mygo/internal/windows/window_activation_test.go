//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"
)

func newActivationFixture(t *testing.T, exStyle uint32) (*window, uintptr) {
	t.Helper()
	hwnd := createWindow(exStyle, "STATIC", "mygo activation fixture", wsPopup, 0, 0, 200, 100, 0)
	if hwnd == 0 {
		t.Fatal("cannot create the fixture window")
	}
	t.Cleanup(func() { procDestroyWindow.Call(hwnd) })
	return &window{hwnd: hwnd}, hwnd
}

func TestSetActivatableTogglesTheNoActivateStyle(t *testing.T) {
	w, hwnd := newActivationFixture(t, 0)
	if windowLong(hwnd, gwlExStyle)&wsExNoActivate != 0 {
		t.Fatal("a plain fixture window came up non-activatable")
	}
	w.SetActivatable(false)
	if windowLong(hwnd, gwlExStyle)&wsExNoActivate == 0 {
		t.Fatal("SetActivatable(false) left the window activatable")
	}
	w.SetActivatable(true)
	if windowLong(hwnd, gwlExStyle)&wsExNoActivate != 0 {
		t.Fatal("SetActivatable(true) left the window non-activatable")
	}
}

func TestAClickMakesANonActivatableWindowActivatable(t *testing.T) {
	w, hwnd := newActivationFixture(t, wsExNoActivate)
	if windowLong(hwnd, gwlExStyle)&wsExNoActivate == 0 {
		t.Fatal("the fixture was not created non-activatable")
	}
	if got := w.onMouseActivate(); got != maActivate {
		t.Fatalf("onMouseActivate() = %v, want maActivate, so a click would not activate the window", got)
	}
	if windowLong(hwnd, gwlExStyle)&wsExNoActivate != 0 {
		t.Fatal("a click on the window left it non-activatable, so the click could not focus it")
	}
}
