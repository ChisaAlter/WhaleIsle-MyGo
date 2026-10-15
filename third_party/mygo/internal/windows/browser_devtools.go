//go:build windows && (amd64 || arm64)

package windows

import "errors"

func (w *window) BrowserDevToolsAvailable() bool {
	// Includes BrowserProfile, exact partition/map generation, live membership,
	// verified dedicated environment, non-isolated preview and native readiness.
	return w.BrowserCookieToolsAvailable()
}

func (w *window) OpenBrowserDevTools() error {
	if !w.BrowserDevToolsAvailable() {
		return errors.New("mygo: inspector requires a ready, verified Browser workspace page")
	}
	// Existing ABI verified again in Microsoft's WebView2 1.0.3537.50 header:
	// ICoreWebView2 C vtable slot 51 is HRESULT OpenDevToolsWindow(void).
	// AreDevToolsEnabled controls context-menu/keyboard entrypoints. Keep it
	// false; this is the separate, explicit host action documented below.
	// https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2?view=webview2-1.0.3537.50#opendevtoolswindow
	// https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2settings?view=webview2-1.0.3537.50#get_aredevtoolsenabled
	// Never enable settings, scripts, host objects, bridge messages or a remote
	// debugging endpoint if the runtime/policy rejects this operation.
	if hr := comCall(w.webview, wvOpenDevToolsWindow); failed(hr) {
		return hresultError("OpenDevToolsWindow", hr)
	}
	// The API is idempotent if already open. WebView2 does not provide a matching
	// close/is-open API, so don't set the legacy sticky devTools boolean.
	return nil
}
