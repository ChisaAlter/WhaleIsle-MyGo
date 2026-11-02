//go:build windows

package pet

import (
	"syscall"
	"testing"

	"github.com/egoist/mygo"
)

func realPetMouseStyle(t *testing.T, w *mygo.Window, wantIgnoring bool) {
	t.Helper()
	proc := syscall.NewLazyDLL("user32.dll").NewProc("GetWindowLongW")
	// GWL_EXSTYLE is -20; use the pointer-sized two's complement for Call.
	style, _, _ := proc.Call(w.NativeHandle(), ^uintptr(19))
	const wsExTransparent = 0x20
	if (style&wsExTransparent != 0) != wantIgnoring {
		t.Errorf("real Win32 WS_EX_TRANSPARENT=%v want=%v style=%x", style&wsExTransparent != 0, wantIgnoring, style)
	}
	realPetEvidence[map[bool]string{true: "blankExStyle", false: "islandExStyle"}[wantIgnoring]] = style
}
