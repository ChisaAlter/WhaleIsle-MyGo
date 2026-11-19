//go:build windows && (amd64 || arm64) && !release

package windows

import (
	"fmt"
	"runtime/debug"
	"syscall"
	"unsafe"
)

// RaiseTrace, when set, receives every call this package makes that asks the
// shell to raise or focus a window. The class fix lives one layer up in Window,
// so a raise recorded here is one that fix cannot see.
var RaiseTrace func(call string, stack []byte)

func traceRaise(call string, hwnd uintptr) {
	if RaiseTrace == nil {
		return
	}
	fg, _, _ := procGetForegroundWindow.Call()
	RaiseTrace(fmt.Sprintf("%s target=0x%04X fg=0x%04X target-chain=%s fg-chain=%s",
		call, hwnd, fg, windowChain(hwnd), windowChain(fg)), debug.Stack())
}

var (
	procTraceGetParent    = user32.NewProc("GetParent")
	procTraceGetWindow    = user32.NewProc("GetWindow")
	procTraceGetClassName = user32.NewProc("GetClassNameW")
)

const gwOwner = 4

// windowChain names a window and walks up through its parent and owner, because
// a raise logged on a child window says nothing about which window the operator
// saw take the foreground.
func windowChain(hwnd uintptr) string {
	chain := ""
	for depth := 0; depth < 4 && hwnd != 0; depth++ {
		chain += fmt.Sprintf("0x%04X[%s]<-", hwnd, traceClassName(hwnd))
		parent, _, _ := procTraceGetParent.Call(hwnd)
		if parent == 0 {
			parent, _, _ = procTraceGetWindow.Call(hwnd, gwOwner)
		}
		hwnd = parent
	}
	return chain
}

func traceClassName(hwnd uintptr) string {
	var buf [256]uint16
	n, _, _ := procTraceGetClassName.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return "?"
	}
	return syscall.UTF16ToString(buf[:n])
}

// activationCause names how a WM_ACTIVATE arrived, because "the window came
// forward" and "a click on the window brought it forward" are different events
// for this fix and only the first one is a defect.
func activationCause(wp uint16) string {
	switch wp {
	case waActive:
		return "window.onWMActivate.programmatic"
	case waClickActive:
		return "window.onWMActivate.click"
	case waInactive:
		return "window.onWMActivate.inactive"
	}
	return ""
}
