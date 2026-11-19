//go:build windows && (amd64 || arm64)

package windows

// applyClientFocus answers a client's request to focus an element. It moves the
// keyboard focus into our surface only when the shell already has that window in
// front, and reports whether it moved.
//
// A client asking a background window's element for focus is driving an
// application the operator is not using. Moving focus there is not a local
// rearrangement: Windows promotes the child that receives focus across a process
// boundary, so the window comes forward and the operator loses what they were
// doing. That is how a headless driver takes the foreground without sending a
// keystroke, and it is the app-side half of the focus theft this build is being
// fixed for. Leaving the shell's focus alone while still reporting the element
// as focused serves the caller with a right to it: a real assistive technology
// drives the foreground window, so its path never reaches this branch.
func applyClientFocus(hwnd uintptr) bool {
	if !focusOnForegroundWindow(hwnd) {
		traceRaise("surface_access.SetFocus.skipped", hwnd)
		return false
	}
	traceRaise("surface_access.SetFocus", hwnd)
	procSetFocus.Call(hwnd)
	return true
}

// focusOnForegroundWindow reports whether hwnd belongs to the window the shell
// currently has in front.
func focusOnForegroundWindow(hwnd uintptr) bool {
	foreground, _, _ := procGetForegroundWindow.Call()
	return focusFollowsForeground(hwnd, foreground)
}

func focusFollowsForeground(hwnd, foreground uintptr) bool {
	if hwnd == 0 || foreground == 0 {
		return false
	}
	root, _, _ := procGetAncestor.Call(hwnd, gaRoot)
	if root == 0 {
		root = hwnd
	}
	return root == foreground
}
