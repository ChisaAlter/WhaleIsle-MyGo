package fake

// SetDevToolsEnabled records the runtime inspector switch for tests.
func (w *Window) SetDevToolsEnabled(enabled bool) {
	w.mu.Lock()
	w.devToolsEnabled = &enabled
	w.mu.Unlock()
}

// DevToolsEnabled reports the last switch value, or nil when never set.
func (w *Window) DevToolsEnabled() *bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.devToolsEnabled
}
