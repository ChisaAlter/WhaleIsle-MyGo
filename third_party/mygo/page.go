package mygo

// Page is the web page a window shows: what it loads, the scripts it runs,
// its developer tools and its events. A window showing native UI
// (WindowOptions.Content) has none.
//
//	win := mygo.NewWindow(mygo.WindowOptions{Title: "Editor", URL: "/"})
//	win.Page().OnDOMReady(func() { log.Println("ready") })
//
// Like the window's, its methods are safe from any goroutine.
type Page struct{ w *Window }

// Page returns the web page the window shows, or nil for a window that
// shows native UI.
func (w *Window) Page() *Page {
	if w.content != nil {
		return nil
	}
	return w.pg
}

// Window returns the window that shows the page.
func (p *Page) Window() *Window { return p.w }

// PageOptions configure the web page of a window.
type PageOptions struct {
	// BrowserProfile opts into the app's persistent Browser-only native data
	// store. Pages with the same BrowserPartition share it; different
	// partitions and default-profile windows remain isolated. Existing cookies
	// and storage are not migrated or deleted, so users
	// must sign in again on first use. Native bindings, preloads and automatic
	// DevTools entrypoints are disabled. Only explicit host OpenBrowserDevTools
	// may open this page's inspector where supported. Requires WebView2 on Windows,
	// macOS 14+, or WebKitGTK 2.16+ on Linux. Incompatible with local/isolated previews.
	BrowserProfile bool
	// BrowserPartition is a required stable, opaque lowercase SHA-256 key for
	// BrowserProfile. It must identify the trusted Host/workspace scope, never a
	// credential, port or random launch ID. It is not a filesystem path.
	BrowserPartition string
	// LocalPDFURL permits one exact loopback PDF capability in an otherwise
	// isolated preview. No external resources, scripts, popups, permissions,
	// downloads or bindings are allowed. Windows only; never accepts file:.
	// LocalPDFViewerURL enables only the product-owned offline reader in an isolated profile.
	LocalPDFViewerURL string
	LocalPDFURL       string
	// LocalHTMLURL permits one exact loopback HTML capability, mutually exclusive
	// with LocalPDFURL. The caller must wrap content in an opaque sandbox with a
	// restrictive CSP. Scripts, network resources, native bindings,
	// permissions, popups, downloads, preloads and developer tools are disabled.
	// Windows only; the URL must be http://127.0.0.1:port/<64hex>/document.html.
	LocalHTMLURL string
	// LocalMediaURL permits a fixed, script-free player and its one derived media
	// resource. Mutually exclusive with PDF/HTML; Windows isolated profile only.
	LocalMediaURL string
	// IsolatedPreview requests a dedicated temporary WebView2 environment on Windows, with
	// native bindings and resource/navigation/permission/download access denied.
	// Experimental platform prerequisite only: it is NOT a renderer-readiness
	// signal. Callers must still use a strict opaque sandbox and validated capture
	// lifecycle. macOS/Linux support only a nonpersistent about:blank encoder with
	// fixed CSP; arbitrary LoadHTML and all Local* preview URLs remain unsupported.
	// Ordinary windows are unchanged.
	IsolatedPreview bool
	// WindowControlsOnly keeps DOM-ready and CSS drag regions, but rejects
	// application calls, events, drops and all other native bridge messages.
	// DisableBindings takes precedence and disables the bridge entirely.
	WindowControlsOnly bool
	// DisableBindings isolates untrusted browsing: no Go calls or app events,
	// even for origins otherwise trusted by the application.
	DisableBindings bool
	// PreloadScript is JavaScript injected into every page before the
	// page's own scripts, after window.mygo is available.
	PreloadScript string
	// TrustedOrigins lists extra origins, such as "https://example.com",
	// whose pages may call bound Go methods. By default only the app's own
	// content can: custom schemes registered with Protocol, file: and
	// about: pages, and loopback dev servers during development. "*"
	// trusts every origin.
	TrustedOrigins []string
	// DevTools controls the web inspector, and for a window showing
	// Content, the inspector of its native UI.
	DevTools DevTools
	// ZoomFactor of the page (default 1).
	ZoomFactor float64
	// UserAgent overrides the user agent string.
	UserAgent string
}
