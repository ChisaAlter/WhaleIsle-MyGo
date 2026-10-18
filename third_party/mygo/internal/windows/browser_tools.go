//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/json"
	"errors"
)

// Reuse the existing ICoreWebView2.CallDevToolsProtocolMethod implementation in
// webview.go. No new COM interface, vtable slot, GUID or JS binding is added.
// CDP definitions: github.com/ChromeDevTools/devtools-protocol,
// Emulation.setEmulatedMedia and Network.clearBrowserCache.
func (w *window) BrowserToolsAvailable() bool {
	return !w.closed && w.ready && w.webview != 0 && !w.opts.IsolatedPreview
}
func (w *window) SetBrowserColorScheme(scheme string, done func(error)) {
	if !w.BrowserToolsAvailable() {
		done(errors.New("mygo: browser page is not ready for appearance controls"))
		return
	}
	value := scheme
	switch scheme {
	case "system":
		value = ""
	case "light", "dark":
	default:
		done(errors.New("mygo: invalid browser color scheme"))
		return
	}
	params, _ := json.Marshal(map[string]any{"features": []map[string]string{{"name": "prefers-color-scheme", "value": value}}})
	w.devtools("Emulation.setEmulatedMedia", string(params), func(_ string, err error) { done(err) })
}
func (w *window) ClearBrowserHTTPCache(done func(error)) {
	if !w.BrowserToolsAvailable() || (w.opts.BrowserProfile && !w.BrowserCookieToolsAvailable()) {
		done(errors.New("mygo: browser page is not ready for cache controls"))
		return
	}
	w.devtools("Network.clearBrowserCache", "{}", func(_ string, err error) { done(err) })
}
