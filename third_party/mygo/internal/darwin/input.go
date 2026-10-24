//go:build darwin

package darwin

import "github.com/egoist/mygo/internal/platform"

// SetBeforeInput stores the key-input callback; see platform.SetBeforeInput.
// WKWebView exposes no pre-handling key event without injecting script into the
// page, so the callback is kept for API parity and never runs
// (Page.OnBeforeInput reports false).
func (w *window) SetBeforeInput(fn func(platform.KeyInput) bool) { w.beforeInput = fn }
