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
	// LocalPDFURL permits one exact loopback PDF capability in an otherwise
	// isolated preview. No external resources, scripts, popups, permissions,
	// downloads or bindings are allowed. Windows only; never accepts file:.
	LocalPDFURL string
	// IsolatedPreview requests a dedicated temporary WebView2 environment with
	// native bindings and resource/navigation/permission/download access denied.
	// Experimental platform prerequisite only: it is NOT a renderer-readiness
	// signal. Callers must still use a strict opaque sandbox and validated capture
	// lifecycle. Unsupported platforms fail closed. Ordinary windows are unchanged.
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
