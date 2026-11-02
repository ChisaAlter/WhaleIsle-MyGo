//go:build windows

package pet

import (
	"math"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// Pet-only native geometry is in physical desktop pixels. MyGo divides each
// display origin by its own DPI but SetBounds multiplies by the current HWND's
// DPI, which is not a consistent desktop space on mixed-DPI monitors.
var petUser32 = syscall.NewLazyDLL("user32.dll")
var petMonitorInfo = petUser32.NewProc("GetMonitorInfoW")
var petGetWindowRect = petUser32.NewProc("GetWindowRect")
var petGetClientRect = petUser32.NewProc("GetClientRect")
var petClientToScreen = petUser32.NewProc("ClientToScreen")
var petSetWindowPos = petUser32.NewProc("SetWindowPos")
var petGetCursorPos = petUser32.NewProc("GetCursorPos")
var petGetDpi = petUser32.NewProc("GetDpiForWindow")

type petNativeRect struct{ Left, Top, Right, Bottom int32 }
type petNativePoint struct{ X, Y int32 }

func petNativeRectangle(r petNativeRect) mygo.Rectangle {
	return mygo.Rectangle{X: int(r.Left), Y: int(r.Top), Width: int(r.Right - r.Left), Height: int(r.Bottom - r.Top)}
}
func platformPetDisplays() []mygo.Display {
	displays := mygo.Screen.Displays()
	for i := range displays {
		var info struct {
			Size          uint32
			Monitor, Work petNativeRect
			Flags         uint32
			Device        [32]uint16
		}
		info.Size = uint32(unsafe.Sizeof(info))
		if ok, _, _ := petMonitorInfo.Call(uintptr(displays[i].ID), uintptr(unsafe.Pointer(&info))); ok != 0 {
			displays[i].Bounds = petNativeRectangle(info.Monitor)
			displays[i].WorkArea = petNativeRectangle(info.Work)
		}
	}
	return displays
}
func petCursorPoint() mygo.Point {
	var p petNativePoint
	petGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return mygo.Point{X: int(p.X), Y: int(p.Y)}
}
func petWindowBounds(w *mygo.Window) mygo.Rectangle {
	var r petNativeRect
	petGetWindowRect.Call(w.NativeHandle(), uintptr(unsafe.Pointer(&r)))
	return petNativeRectangle(r)
}
func petContentBounds(w *mygo.Window) mygo.Rectangle {
	var r petNativeRect
	var p petNativePoint
	petGetClientRect.Call(w.NativeHandle(), uintptr(unsafe.Pointer(&r)))
	petClientToScreen.Call(w.NativeHandle(), uintptr(unsafe.Pointer(&p)))
	return mygo.Rectangle{X: int(p.X), Y: int(p.Y), Width: int(r.Right - r.Left), Height: int(r.Bottom - r.Top)}
}
func petWindowScale(w *mygo.Window) float64 {
	dpi, _, _ := petGetDpi.Call(w.NativeHandle())
	if dpi == 0 {
		return 1
	}
	return float64(dpi) / 96
}
func petPlaceNative(w *mygo.Window, b mygo.Rectangle) {
	if petWindowBounds(w) == b {
		return
	}
	// DisableMove is also enforced for programmatic SetWindowPos by MyGo's
	// WM_WINDOWPOSCHANGING handler. Temporarily permit this synchronous owned
	// placement (no input dispatch), then restore the user's move restriction.
	movable := w.IsMovable()
	if !movable {
		w.SetMovable(true)
		defer w.SetMovable(false)
	}
	// Native DPI change may apply a suggested rectangle; reapply our exact
	// display bounds once after that synchronous notification has completed.
	for i := 0; i < 2; i++ {
		petSetWindowPos.Call(w.NativeHandle(), 0, uintptr(b.X), uintptr(b.Y), uintptr(b.Width), uintptr(b.Height), 0x14)
	} // NOZORDER|NOACTIVATE
}
func petCoordinateScale(d mygo.Display) float64 {
	if d.ScaleFactor <= 0 {
		return 1
	}
	return d.ScaleFactor
}
func petStoredPosition(position motionPoint) *config.PetPosition {
	return &config.PetPosition{X: int(math.Round(position.x)), Y: int(math.Round(position.y)), Space: config.PetPositionWindowsPhysical}
}
func petRestorePosition(saved *config.PetPosition, displays []mygo.Display) *config.PetPosition {
	if saved == nil {
		return nil
	}
	if saved.Space == config.PetPositionWindowsPhysical {
		copy := *saved
		return &copy
	}
	// Empty space is the historical toolkit coordinate system. Reconstruct
	// its per-monitor DIP bounds and convert with that monitor's own scale.
	for _, d := range displays {
		s := petCoordinateScale(d)
		b := d.Bounds
		x, y := float64(saved.X), float64(saved.Y)
		if x >= float64(b.X)/s && x < float64(b.X+b.Width)/s && y >= float64(b.Y)/s && y < float64(b.Y+b.Height)/s {
			return petStoredPosition(motionPoint{x: x * s, y: y * s})
		}
	}
	return nil
}
