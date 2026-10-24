//go:build linux

package linux

// SetDevToolsEnabled keeps the runtime inspector switch; WebKitGTK's
// enable-developer-extras is not wired here yet, so the value is stored only.
func (w *window) SetDevToolsEnabled(enabled bool) { w.devToolsEnabled = &enabled }
