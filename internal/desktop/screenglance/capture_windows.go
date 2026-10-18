//go:build windows

package screenglance

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	gdi32             = syscall.NewLazyDLL("gdi32.dll")
	isWindow          = user32.NewProc("IsWindow")
	monitorFromWindow = user32.NewProc("MonitorFromWindow")
	getMonitorInfo    = user32.NewProc("GetMonitorInfoW")
	setThreadDPI      = user32.NewProc("SetThreadDpiAwarenessContext")
	getDC             = user32.NewProc("GetDC")
	releaseDC         = user32.NewProc("ReleaseDC")
	createDC          = gdi32.NewProc("CreateCompatibleDC")
	deleteDC          = gdi32.NewProc("DeleteDC")
	createDIB         = gdi32.NewProc("CreateDIBSection")
	selectObject      = gdi32.NewProc("SelectObject")
	deleteObject      = gdi32.NewProc("DeleteObject")
	setStretchMode    = gdi32.NewProc("SetStretchBltMode")
	stretchBlt        = gdi32.NewProc("StretchBlt")
	gdiFlush          = gdi32.NewProc("GdiFlush")
)

type rect struct{ Left, Top, Right, Bottom int32 }
type monitorInfo struct {
	Size          uint32
	Monitor, Work rect
	Flags         uint32
}
type bitmapHeader struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}
type bitmapInfo struct {
	Header bitmapHeader
	Colors [1]uint32
}

// Capture reads only the monitor containing the specified existing pet window.
// Pixels are downscaled directly into a <=768x480 memory bitmap, encoded JPEG70,
// and never written to disk. The caller must obtain approval before calling.
func Capture(ctx context.Context, window uintptr) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if window == 0 {
		return nil, fmt.Errorf("screen glance: missing pet window")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Scope physical-coordinate awareness to this capture's OS thread.
	if setThreadDPI.Find() != nil {
		return nil, fmt.Errorf("screen glance: physical-coordinate capture unavailable")
	}
	old, _, _ := setThreadDPI.Call(^uintptr(3)) // PER_MONITOR_AWARE_V2 (-4)
	if old == 0 {
		return nil, fmt.Errorf("screen glance: physical-coordinate capture unavailable")
	}
	defer setThreadDPI.Call(old)
	if ok, _, _ := isWindow.Call(window); ok == 0 {
		return nil, fmt.Errorf("screen glance: pet window closed")
	}
	monitor, _, _ := monitorFromWindow.Call(window, 0) // no unrelated-display fallback
	if monitor == 0 {
		return nil, fmt.Errorf("screen glance: pet display unavailable")
	}
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, _ := getMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return nil, fmt.Errorf("screen glance: monitor bounds unavailable")
	}
	width, height := int(info.Monitor.Right-info.Monitor.Left), int(info.Monitor.Bottom-info.Monitor.Top)
	w, h, err := thumbnailSize(width, height)
	if err != nil {
		return nil, err
	}
	screen, _, _ := getDC.Call(0)
	if screen == 0 {
		return nil, fmt.Errorf("screen glance: display context unavailable")
	}
	defer releaseDC.Call(0, screen)
	memory, _, _ := createDC.Call(screen)
	if memory == 0 {
		return nil, fmt.Errorf("screen glance: memory context unavailable")
	}
	defer deleteDC.Call(memory)
	bi := bitmapInfo{Header: bitmapHeader{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}}
	var bits unsafe.Pointer
	bitmap, _, _ := createDIB.Call(screen, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 {
		return nil, fmt.Errorf("screen glance: frame allocation failed")
	}
	defer deleteObject.Call(bitmap)
	if bits == nil {
		return nil, fmt.Errorf("screen glance: frame pixels unavailable")
	}
	previous, _, _ := selectObject.Call(memory, bitmap)
	if previous == 0 || previous == ^uintptr(0) {
		return nil, fmt.Errorf("screen glance: frame selection failed")
	}
	defer selectObject.Call(memory, previous)
	setStretchMode.Call(memory, 3) // COLORONCOLOR
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const sourceCopy = 0x00CC0020 | 0x40000000 // SRCCOPY | CAPTUREBLT
	ok, _, _ := stretchBlt.Call(memory, 0, 0, uintptr(w), uintptr(h), screen,
		uintptr(info.Monitor.Left), uintptr(info.Monitor.Top), uintptr(width), uintptr(height), sourceCopy)
	if ok == 0 {
		return nil, fmt.Errorf("screen glance: capture failed")
	}
	// Direct access to DIB pixels must wait for this thread's queued GDI writes.
	if ok, _, _ := gdiFlush.Call(); ok == 0 {
		return nil, fmt.Errorf("screen glance: frame read failed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pixels := unsafe.Slice((*byte)(bits), w*h*4)
	result, err := encodeBGRA(pixels, w, h)
	clear(pixels)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		clear(result)
		return nil, err
	}
	return result, nil
}
