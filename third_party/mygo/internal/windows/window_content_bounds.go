//go:build windows && (amd64 || arm64)

package windows

import (
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// clientPixelInsets measures actual non-client pixels, including the single
// Windows 11 hidden-titlebar pixel and the invisible resize borders. It also
// handles native captions without reimplementing WM_NCCALCSIZE policy.
func (w *window) clientPixelInsets() (rect, bool) {
	var outer, client rect
	var origin point
	if ok, _, _ := procGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&outer))); ok == 0 {
		return rect{}, false
	}
	if ok, _, _ := procGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return rect{}, false
	}
	if ok, _, _ := procClientToScreen.Call(w.hwnd, uintptr(unsafe.Pointer(&origin))); ok == 0 {
		return rect{}, false
	}
	return rect{origin.X - outer.Left, origin.Y - outer.Top, outer.Right - origin.X - (client.Right - client.Left), outer.Bottom - origin.Y - (client.Bottom - client.Top)}, true
}

// contentOuterPixels scales the desired client size independently of its
// position, then expands by measured physical insets. Scaling absolute far
// edges would let an odd/fractional-DPI position change the requested size.
func contentOuterPixels(r platform.Rect, dpi int, insets rect) rect {
	x, y := toPx(r.X, dpi), toPx(r.Y, dpi)
	return rect{x - insets.Left, y - insets.Top, x + toPx(r.Width, dpi) + insets.Right, y + toPx(r.Height, dpi) + insets.Bottom}
}
