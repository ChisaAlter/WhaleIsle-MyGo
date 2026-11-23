//go:build windows && (amd64 || arm64)

package windows

import "sync/atomic"

// displayEpoch counts the changes of the displays and of the desktop's
// settings (WM_DISPLAYCHANGE, WM_SETTINGCHANGE), after which the work
// areas windows noted are read again.
var displayEpoch atomic.Uint64

// workArea is the orientation of the work area of the monitor showing a
// window, as the window last read it, for epoch, until the window moves or
// changes DPI.
type workArea struct {
	valid, landscape, known bool
	epoch                   uint64
}

// WorkAreaLandscape reports whether the work area of the monitor showing
// the surface's window is wider than high, and whether that is known: not
// for a window on no monitor, or a square work area. It reads the monitor
// once until the window moves, changes DPI, or the displays or the
// desktop's settings change (platform.WorkAreaSurface).
func (s *surface) WorkAreaLandscape() (landscape, known bool) {
	w := s.w
	a := &w.work
	if epoch := displayEpoch.Load(); !a.valid || a.epoch != epoch {
		*a = workArea{valid: true, epoch: epoch}
		if mon, _, _ := procMonitorFromWindow.Call(w.hwnd, 0); mon != 0 { // MONITOR_DEFAULTTONULL
			r := monitorWorkArea(mon)
			if width, height := r.Right-r.Left, r.Bottom-r.Top; width > 0 && height > 0 && width != height {
				a.landscape, a.known = width > height, true
			}
		}
	}
	return a.landscape, a.known
}
