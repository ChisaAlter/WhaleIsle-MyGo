//go:build linux

package linux

import "github.com/egoist/mygo/internal/platform"

// SetBeforeInput stores the key-input callback; see platform.SetBeforeInput.
// WebKitGTK exposes no pre-handling key event for a web view, so the callback
// is kept for API parity and never runs (Page.OnBeforeInput reports false).
func (w *window) SetBeforeInput(fn func(platform.KeyInput) bool) { w.beforeInput = fn }
