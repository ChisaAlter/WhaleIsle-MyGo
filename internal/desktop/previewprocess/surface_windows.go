//go:build windows && (amd64 || arm64)

package previewprocess

import (
	"errors"
	"golang.org/x/sys/windows"
	"math"
	"unsafe"
)

var surfaceUser = windows.NewLazySystemDLL("user32.dll")
var surfacePID = surfaceUser.NewProc("GetWindowThreadProcessId")
var surfaceParent = surfaceUser.NewProc("SetParent")
var surfaceGetParent = surfaceUser.NewProc("GetParent")
var surfaceStyle = surfaceUser.NewProc("SetWindowLongPtrW")
var surfacePosition = surfaceUser.NewProc("SetWindowPos")
var surfaceShow = surfaceUser.NewProc("ShowWindow")
var surfaceDPI = surfaceUser.NewProc("GetDpiForWindow")
var surfaceRegion = surfaceUser.NewProc("SetWindowRgn")
var surfaceGDI = windows.NewLazySystemDLL("gdi32.dll")
var surfaceRect = surfaceGDI.NewProc("CreateRectRgn")
var surfaceDelete = surfaceGDI.NewProc("DeleteObject")

func validSurfaceWindow(hwnd uintptr, pid uint32) bool {
	var owner uint32
	result, _, _ := surfacePID.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
	return hwnd != 0 && result != 0 && owner == pid
}

// PresentSurface embeds only the HWND verified against this helper's PID.
// The parent HWND stays in the trusted process; no parent/window capability is
// sent to the untrusted document. Repositioning preserves the iframe and JS state.
func (s *Session) PresentSurface(parent uintptr, x, y, w, h, cx, cy, cw, ch int) error {
	s.surfaceMu.Lock()
	defer s.surfaceMu.Unlock()
	select {
	case <-s.Done():
		return errors.New("preview closed")
	default:
	}
	hwnd := s.surfaceHandle
	if !validSurfaceWindow(hwnd, s.surfacePID) {
		return errors.New("preview surface owner changed")
	}
	if parent == 0 || w <= 0 || h <= 0 || cw <= 0 || ch <= 0 {
		surfaceShow.Call(hwnd, 0)
		return nil
	}
	owner, _, _ := surfaceGetParent.Call(hwnd)
	if owner != parent {
		// WS_CHILD | WS_CLIPSIBLINGS | WS_CLIPCHILDREN, without popup chrome.
		styleIndex := int32(-16)
		surfaceStyle.Call(hwnd, uintptr(styleIndex), 0x46000000)
		surfaceParent.Call(hwnd, parent)
		owner, _, _ = surfaceGetParent.Call(hwnd)
		if owner != parent {
			return errors.New("could not attach isolated preview surface")
		}
	}
	dpi, _, _ := surfaceDPI.Call(parent)
	if dpi == 0 {
		dpi = 96
	}
	scale := float64(dpi) / 96
	px := func(v int) uintptr { return uintptr(int32(math.Round(float64(v) * scale))) }
	// Clip against the actual native UI paint clip, not just the card bounds.
	left, top := max(0, cx-x), max(0, cy-y)
	right, bottom := min(w, cx+cw-x), min(h, cy+ch-y)
	if right <= left || bottom <= top {
		surfaceShow.Call(hwnd, 0)
		return nil
	}
	region, _, _ := surfaceRect.Call(px(left), px(top), px(right), px(bottom))
	if region == 0 {
		return errors.New("could not clip preview surface")
	}
	ok, _, _ := surfaceRegion.Call(hwnd, region, 1)
	if ok == 0 {
		surfaceDelete.Call(region)
		return errors.New("could not install preview clip")
	}
	// NOACTIVATE | SHOWWINDOW | ASYNCWINDOWPOS | NOZORDER | FRAMECHANGED.
	ok, _, _ = surfacePosition.Call(hwnd, 0, px(x), px(y), px(w), px(h), 0x4074)
	if ok == 0 {
		return errors.New("could not position preview surface")
	}
	return nil
}
