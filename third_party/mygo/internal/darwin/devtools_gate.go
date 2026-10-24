//go:build darwin

package darwin

// SetDevToolsEnabled keeps the runtime inspector switch; WKWebView's
// isInspectable is not exposed here yet, so the value is stored only.
func (w *window) SetDevToolsEnabled(enabled bool) { w.devToolsEnabled = &enabled }
