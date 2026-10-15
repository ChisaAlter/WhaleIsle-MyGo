//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"unsafe"
)

// ABI verified by counting the complete repr(C) ICoreWebView2_Vtbl in
// microsoft/windows-rs at c4932863208881f0cf9270471864d94761d09cd0,
// crates/libs/webview/src/bindings.rs:555-660 (IUnknown contributes 3 slots).
// Microsoft Learn confirms HRESULT get_BrowserProcessId(UINT32*).
// This identity observation never attests to token/job isolation.
const previewBrowserPIDSlot = 37

// BrowserProcessID observes the native identity after WebView2 initialization.
func (w *window) BrowserProcessID() (uint32, error) {
	if w.webview == 0 { return 0, errors.New("WebView2 is not initialized") }
	var pid uint32
	hr := comCall(w.webview, previewBrowserPIDSlot, uintptr(unsafe.Pointer(&pid)))
	if failed(hr) { return 0, hresultError("get_BrowserProcessId", hr) }
	if pid == 0 { return 0, errors.New("WebView2 returned an empty browser process ID") }
	return pid, nil
}
