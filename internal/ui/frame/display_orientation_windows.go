//go:build windows

package frame

import "unsafe"

var (
	procFrameMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procFrameGetMonitorInfo    = user32.NewProc("GetMonitorInfoW")
)

// displayLandscape follows screen.availWidth/availHeight, not a desktop
// window's aspect. viewportDIP has already validated/cached ownWindow; when no
// such owned visible window exists (headless tests), screen data is unknown.
// Resolve the monitor each time so a move to another display takes effect.
func displayLandscape() (landscape, known bool) {
	ownWindowMu.Lock()
	defer ownWindowMu.Unlock()
	if ownWindow == 0 {
		return false, false
	}
	monitor, _, _ := procFrameMonitorFromWindow.Call(ownWindow, 0)
	if monitor == 0 {
		return false, false
	}
	var info struct {
		Size          uint32
		Monitor, Work winRect
		Flags         uint32
	}
	info.Size = uint32(unsafe.Sizeof(info))
	if ok, _, _ := procFrameGetMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return false, false
	}
	width, height := info.Work.Right-info.Work.Left, info.Work.Bottom-info.Work.Top
	if width <= 0 || height <= 0 || width == height {
		return false, false
	}
	return width > height, true
}
