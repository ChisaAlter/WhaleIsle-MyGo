//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"testing"
)

func TestFocusFollowsForegroundOnlyForTheForegroundRoot(t *testing.T) {
	cases := []struct {
		name       string
		hwnd       uintptr
		foreground uintptr
		want       bool
	}{
		{name: "same window", hwnd: 0x1111, foreground: 0x1111, want: true},
		{name: "different window", hwnd: 0x2222, foreground: 0x1111, want: false},
		{name: "no foreground", hwnd: 0x1111, foreground: 0, want: false},
		{name: "no window", hwnd: 0, foreground: 0x1111, want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := focusFollowsForeground(c.hwnd, c.foreground); got != c.want {
				t.Fatalf("focusFollowsForeground(%#x, %#x) = %v, want %v", c.hwnd, c.foreground, got, c.want)
			}
		})
	}
}

// newBackgroundWindow shows a window without activating it, so the test can ask
// for focus on something the shell does not have in front. It never takes the
// foreground, which is why it can run on a developer's machine.
func newBackgroundWindow(t *testing.T) uintptr {
	t.Helper()
	hwnd := createWindow(0, "STATIC", "mygo focus fixture", wsPopup, 0, 0, 200, 100, 0)
	if hwnd == 0 {
		t.Fatal("cannot create the fixture window")
	}
	t.Cleanup(func() { procDestroyWindow.Call(hwnd) })
	procShowWindow.Call(hwnd, swShowNoActivate)
	return hwnd
}

func TestClientFocusLeavesABackgroundWindowWhereItIs(t *testing.T) {
	hwnd := newBackgroundWindow(t)
	if focusOnForegroundWindow(hwnd) {
		t.Skip("fixture window came up in front; the machine has no stable foreground")
	}
	foregroundBefore, _, _ := procGetForegroundWindow.Call()
	if applyClientFocus(hwnd) {
		t.Fatal("a background window moved the keyboard focus into itself")
	}
	foregroundAfter, _, _ := procGetForegroundWindow.Call()
	if foregroundAfter != foregroundBefore {
		t.Fatalf("foreground changed %#x -> %#x while answering a background SetFocus", foregroundBefore, foregroundAfter)
	}
}

func TestClientFocusMovesFocusWhenTheWindowIsAlreadyInFront(t *testing.T) {
	if os.Getenv("MYGO_FOCUS_FOREGROUND_TEST") != "1" {
		t.Skip("set MYGO_FOCUS_FOREGROUND_TEST=1: this test takes the foreground from whoever is using the machine")
	}
	hwnd := createWindow(0, "STATIC", "mygo focus fixture", wsPopup, 0, 0, 200, 100, 0)
	if hwnd == 0 {
		t.Fatal("cannot create the fixture window")
	}
	t.Cleanup(func() { procDestroyWindow.Call(hwnd) })
	procShowWindow.Call(hwnd, swShow)
	procSetForegroundWindow.Call(hwnd)
	if !focusOnForegroundWindow(hwnd) {
		t.Skip("the shell refused to put the fixture in front")
	}
	if !applyClientFocus(hwnd) {
		t.Fatal("an element of the foreground window was refused focus")
	}
}
