//go:build windows

package frame

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procGetClientRect       = user32.NewProc("GetClientRect")
	procGetDpiForWindow     = user32.NewProc("GetDpiForWindow")
	procGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	procGetCurrentProcessID = kernel32.NewProc("GetCurrentProcessId")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")

	ownWindowMu sync.Mutex
	ownWindow   uintptr

	// One trampoline for the process: syscall.NewCallback allocates a new
	// callback each call, and this runs every frame.
	enumCallbackOnce sync.Once
	enumCallback     uintptr
	enumPid          uintptr
	enumBest         uintptr
	enumBestTitled   uintptr
	enumBestArea     float32
	enumTitledArea   float32
)

type winRect struct{ Left, Top, Right, Bottom int32 }

// windowPixels reports the device-pixel client size and DPI of the process's
// main visible window, or zeros when there is none (the headless tester, other
// platforms). The handle is revalidated every call: a process may own several
// top-level windows and the one that was largest at startup may since have been
// hidden while the real window was resized.
func windowPixels() (width, height float32, dpi uint32) {
	ownWindowMu.Lock()
	defer ownWindowMu.Unlock()
	if w, h, d, ok := measureWindow(ownWindow); ok {
		return w, h, d
	}
	ownWindow = findOwnWindow()
	if w, h, d, ok := measureWindow(ownWindow); ok {
		return w, h, d
	}
	ownWindow = 0
	return 0, 0, 0
}

// measureWindow reports the client size of hwnd in device pixels, its DPI, and
// whether it is a plausible main window at all.
func measureWindow(hwnd uintptr) (width, height float32, dpi uint32, ok bool) {
	if hwnd == 0 {
		return 0, 0, 0, false
	}
	if visible, _, _ := procIsWindowVisible.Call(hwnd); visible == 0 {
		return 0, 0, 0, false
	}
	var r winRect
	if got, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); got == 0 {
		return 0, 0, 0, false
	}
	w, h := float32(r.Right-r.Left), float32(r.Bottom-r.Top)
	if w < 200 || h < 200 {
		return 0, 0, 0, false
	}
	d, _, _ := procGetDpiForWindow.Call(hwnd)
	return w, h, uint32(d), true
}

// findOwnWindow prefers the process's titled window with the largest client area,
// falling back to the largest untitled one.
func findOwnWindow() uintptr {
	pid, _, _ := procGetCurrentProcessID.Call()
	enumPid = pid
	enumBest, enumBestTitled = 0, 0
	enumBestArea, enumTitledArea = 0, 0
	procEnumWindows.Call(processEnumCallback(), 0)
	if enumBestTitled != 0 {
		return enumBestTitled
	}
	return enumBest
}

// processEnumCallback builds the EnumWindows trampoline once.
func processEnumCallback() uintptr {
	enumCallbackOnce.Do(func() {
		enumCallback = syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
			var owner uint32
			procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
			if uintptr(owner) != enumPid {
				return 1
			}
			w, h, _, ok := measureWindow(hwnd)
			if !ok {
				return 1
			}
			area := w * h
			if area > enumBestArea {
				enumBest, enumBestArea = hwnd, area
			}
			if titleLen, _, _ := procGetWindowTextLength.Call(hwnd); titleLen > 0 && area > enumTitledArea {
				enumBestTitled, enumTitledArea = hwnd, area
			}
			return 1
		})
	})
	return enumCallback
}
