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
// where supported. On macOS it only enables public Safari inspection: the user
// must select this page from Safari's Develop menu. It does not open an in-app
// inspector; close the Browser tab to revoke inspection. It is invoked
// in response to an explicit host UI action. The inspector can inspect and modify
// that page's site data; do not invoke automatically or expose this to web content.
// Ordinary/plugin/isolated-preview pages are rejected. Bindings, script policy,
// permissions and automatic DevTools entrypoints remain unchanged. On Linux,
// developer-extras is scoped to this explicit inspector session and reset by
// native closure, Page.CloseDevTools or destruction; menu/keyboard inspector
// entrypoints stay disabled. The user can close the native inspector window.
// Other backends need not provide a programmatic close/toggle capability.
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
