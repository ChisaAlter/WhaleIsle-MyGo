package fake

import "github.com/egoist/mygo/internal/platform"

// SetBeforeInput stores the key-input callback; see platform.SetBeforeInput.
func (w *Window) SetBeforeInput(fn func(platform.KeyInput) bool) { w.mu.Lock(); w.beforeInput = fn; w.mu.Unlock() }

// FireBeforeInput delivers a key event to the app's callback and reports whether
// it was handled. Tests use it to exercise Page.OnBeforeInput without a GUI.
func (w *Window) FireBeforeInput(in platform.KeyInput) bool {
	w.mu.Lock()
	fn := w.beforeInput
	w.mu.Unlock()
	if fn == nil {
		return false
	}
	return fn(in)
}
