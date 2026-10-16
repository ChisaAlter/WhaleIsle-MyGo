//go:build windows

package webview

import (
	"github.com/egoist/mygo"
	"math"
	"syscall"
)

var (
	occlusionUser    = syscall.NewLazyDLL("user32.dll")
	occlusionGDI     = syscall.NewLazyDLL("gdi32.dll")
	occlusionDPI     = occlusionUser.NewProc("GetDpiForWindow")
	occlusionSet     = occlusionUser.NewProc("SetWindowRgn")
	occlusionRect    = occlusionGDI.NewProc("CreateRectRgn")
	occlusionCombine = occlusionGDI.NewProc("CombineRgn")
	occlusionDelete  = occlusionGDI.NewProc("DeleteObject")
)

func setNativeOcclusions(win *mygo.Window, width, height int, holes []mygo.Rectangle) {
	mygo.RunOnMain(func() {
		if win.IsDestroyed() {
			return
		}
		hwnd := win.NativeHandle()
		if len(holes) == 0 {
			occlusionSet.Call(hwnd, 0, 1)
			return
		}
		dpi, _, _ := occlusionDPI.Call(hwnd)
		if dpi == 0 {
			dpi = 96
		}
		scale := float64(dpi) / 96
		low := func(v int) uintptr { return uintptr(int32(math.Floor(float64(v) * scale))) }
		high := func(v int) uintptr { return uintptr(int32(math.Ceil(float64(v) * scale))) }
		region, _, _ := occlusionRect.Call(0, 0, high(width), high(height))
		if region == 0 {
			return
		}
		for _, r := range holes {
			hole, _, _ := occlusionRect.Call(low(r.X), low(r.Y), high(r.X+r.Width), high(r.Y+r.Height))
			if hole == 0 {
				occlusionDelete.Call(region)
				return
			}
			result, _, _ := occlusionCombine.Call(region, region, hole, 4) // RGN_DIFF
			occlusionDelete.Call(hole)
			if result == 0 {
				occlusionDelete.Call(region)
				return
			}
		}
		result, _, _ := occlusionSet.Call(hwnd, region, 1)
		// Windows takes ownership of a successfully installed region.
		if result == 0 {
			occlusionDelete.Call(region)
		}
	})
}
