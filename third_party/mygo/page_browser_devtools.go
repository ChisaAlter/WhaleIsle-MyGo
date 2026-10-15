package mygo

// BrowserDevToolsAvailable reports the optional explicit Browser-only inspector
// capability. It does not enable automatic shortcuts or context-menu access.
func (p *Page) BrowserDevToolsAvailable() bool {
	return onMainValue(func() bool {
		if p == nil || p.w == nil || p.w.native == nil || p.w.content != nil {
			return false
		}
		return browserDevToolsProvider(p.w.native, p.w.disableBindings) != nil
	})
}

// OpenBrowserDevTools opens the current verified Browser page's native inspector
// in response to an explicit host UI action. The inspector can inspect and modify
// that page's site data; do not invoke automatically or expose this to web content.
// Ordinary/plugin/isolated-preview pages are rejected. Bindings, script policy,
// permissions and automatic DevTools entrypoints remain unchanged. The user closes
// the native inspector window; this API does not invent a close/toggle capability.
func (p *Page) OpenBrowserDevTools() error {
	err := errDestroyed
	onMain(func() {
		if p == nil || p.w == nil || p.w.native == nil || p.w.content != nil {
			return
		}
		err = openBrowserDevTools(p.w.native, p.w.disableBindings)
	})
	return err
}
