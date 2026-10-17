//go:build windows

package webview

import (
	"errors"
	"math"

	"github.com/egoist/mygo"
)

func clipMediaNativeWindow(w *mygo.Window, width, height int, holes []mygo.Rectangle) error {
	// Reuse the ordinary browser's existing user32/GDI entrypoints. No process,
	// token, entitlement, firewall or permission changes are involved.
	hwnd := w.NativeHandle()
	if hwnd == 0 || w.IsDestroyed() {
		return errors.New("media: native window was destroyed")
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
		return errors.New("media: could not allocate the native clip")
	}
	owned := true
	defer func() {
		if owned {
			occlusionDelete.Call(region)
		}
	}()
	for _, r := range holes {
		hole, _, _ := occlusionRect.Call(low(r.X), low(r.Y), high(r.X+r.Width), high(r.Y+r.Height))
		if hole == 0 {
			return errors.New("media: could not allocate a native occlusion")
		}
		ok, _, _ := occlusionCombine.Call(region, region, hole, 4)
		occlusionDelete.Call(hole)
		if ok == 0 {
			return errors.New("media: could not combine native occlusions")
		}
	}
	ok, _, _ := occlusionSet.Call(hwnd, region, 1)
	if ok == 0 {
		return errors.New("media: could not install the native clip")
	}
	owned = false // SetWindowRgn takes ownership only on success.
	return nil
}
