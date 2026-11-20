//go:build windows && (amd64 || arm64)

package windows

import (
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

// The real system, unmodified: whatever the shell has in front, a request against
// a window that is not it must leave the foreground handle alone.
func TestClientFocusLeavesABackgroundWindowWhereItIs(t *testing.T) {
	hwnd := newBackgroundWindow(t)
	if focusOnForegroundWindow(hwnd) {
		t.Skip("fixture window came up in front; the machine has no stable foreground")
	}
	foregroundBefore := shellForeground()
	if applyClientFocus(hwnd) {
		t.Fatal("a background window moved the keyboard focus into itself")
	}
	if foregroundAfter := shellForeground(); foregroundAfter != foregroundBefore {
		t.Fatalf("foreground changed %#x -> %#x while answering a background SetFocus", foregroundBefore, foregroundAfter)
	}
}

// Both branches of the policy, with the two system inputs answered by the test
// instead of by the desktop: a background window is refused and never asked to
// take focus, and the same call on a window the shell has in front moves focus.
func TestClientFocusPolicyDecidesWithoutTouchingTheDesktop(t *testing.T) {
	savedForeground, savedFocus := shellForeground, setKeyboardFocus
	t.Cleanup(func() {
		shellForeground, setKeyboardFocus = savedForeground, savedFocus
	})

	hwnd := newBackgroundWindow(t)
	var moved []uintptr
	setKeyboardFocus = func(target uintptr) { moved = append(moved, target) }

	shellForeground = func() uintptr { return hwnd + 0x1000 }
	if applyClientFocus(hwnd) {
		t.Fatal("a background client was allowed to move the keyboard focus")
	}
	if len(moved) != 0 {
		t.Fatalf("a background client asked the shell to move focus to %#x", moved[0])
	}

	shellForeground = func() uintptr { return hwnd }
	if !applyClientFocus(hwnd) {
		t.Fatal("a client of the foreground window was refused focus")
	}
	if len(moved) != 1 || moved[0] != hwnd {
		t.Fatalf("the foreground branch moved focus to %#x (calls: %d), want %#x once", moved, len(moved), hwnd)
	}
}
