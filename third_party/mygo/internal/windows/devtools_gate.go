//go:build windows

package windows

// SetDevToolsEnabled applies the runtime inspector switch to WebView2's
// AreDevToolsEnabled setting; the built-in Ctrl+Shift+I accelerator checks it,
// so a disabled page cannot open DevTools even though the key still arrives.
func (w *window) SetDevToolsEnabled(enabled bool) {
	w.devToolsGate = &enabled
	if w.settings != 0 {
		comCall(w.settings, setPutAreDevToolsEnabled, boolArg(enabled))
	}
}
