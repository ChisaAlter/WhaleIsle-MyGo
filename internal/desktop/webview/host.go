package webview

import (
	"errors"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/shortcutprefs"
)

// HostOptions configure a borderless WebView2 child window.
type HostOptions struct {
	ShortcutPreferences *shortcutprefs.Store
	// BrowserProfile selects the persistent Browser-only login/storage area.
	// It cannot be combined with an authenticated Host session or local preview.
	BrowserProfile bool
	// BrowserPartition is a stable opaque Host/workspace digest, never a path.
	BrowserPartition string
	// LocalPDFURL is a single read-only local PDF capability. It uses the
	// dedicated isolated WebView profile and exact-resource deny-by-default policy.
	LocalPDFViewerURL string
	LocalPDFURL       string
	// LocalHTMLURL is one sandboxed local HTML capability. It is mutually
	// exclusive with PDF and Host sessions and uses a private preview profile.
	LocalHTMLURL string
	// LocalMediaURL is a fixed player plus exactly one derived media route.
	LocalMediaURL string
	// OnClosed releases the local document lease, including creation failures.
	OnClosed func()

	// SessionOrigin/SessionCookie reuse one exact loopback Host session through a
	// temporary transport. This always disables all native bindings.
	SessionOrigin string
	SessionCookie string
	// Untrusted disables all native bindings and app events for browsing.
	Untrusted bool
	OnChange  func()
	OnFocus   func()
	// Parent is the main Content window the host stays above.
	Parent *mygo.Window
	// URL is loaded on create (http/https only).
	URL string
	// Title shown in the OS window list (frameless chrome is hidden).
	Title string
	// TopInset DIPs reserved above the webview (main title bar + panel chrome).
	TopInset int
	// TrustedOrigins passed to PageOptions (defaults to the URL origin).
	TrustedOrigins []string
}

// Host is a MyGo URL window (WebView2 on Windows) owned by the main window.
// Native ui.Content cannot embed a webview; this child covers the panel area.
type Host struct {
	viewport          *mygo.Rectangle
	localPDFViewerURL string
	localPDFURL       string
	localHTMLURL      string
	localMediaURL     string
	onClosed          func()

	gateway         *pluginslots.Gateway
	mu              sync.Mutex
	parent          *mygo.Window
	win             *mygo.Window
	topInset        int
	url             string
	offResize       func()
	offMove         func()
	navigationEpoch uint64
	capturePending  bool
	loading         bool
	stopped         bool
	err             string
	title           string
	changed         func()
}

// NewHost creates a hidden frameless child WebView2 window. Call
// SyncBounds then SetVisible(true) after the parent has a real size.
func NewHost(opts HostOptions) (*Host, error) {
	localPreview := opts.LocalPDFViewerURL != "" || opts.LocalPDFURL != "" || opts.LocalHTMLURL != "" || opts.LocalMediaURL != ""
	if opts.BrowserPartition != "" && !opts.BrowserProfile {
		return nil, errors.New("Browser partition requires a dedicated Browser profile")
	}
	if opts.BrowserProfile {
		if len(opts.BrowserPartition) != 64 || strings.Trim(opts.BrowserPartition, "0123456789abcdef") != "" {
			return nil, errors.New("Browser requires a stable workspace partition")
		}
		if localPreview || opts.SessionOrigin != "" || opts.SessionCookie != "" {
			return nil, errors.New("Browser profile cannot use local previews or Host sessions")
		}
		opts.Untrusted = true
	}
	created := false
	if localPreview {
		// The lease is released exactly once, even if creation fails before a
		// Host exists or native destruction follows a load/process failure.
		callback := opts.OnClosed
		var closeOnce sync.Once
		opts.OnClosed = func() {
			closeOnce.Do(func() {
				if callback != nil {
					callback()
				}
			})
		}
		defer func() {
			if !created {
				opts.OnClosed()
			}
		}()
	}
	if opts.Parent == nil {
		return nil, errors.New("webview: parent window is required")
	}
	raw := strings.TrimSpace(opts.URL)
	if raw == "" {
		return nil, errors.New("webview: empty URL")
	}
	if err := validateHTTPURL(raw); err != nil {
		return nil, err
	}
	if localPreview && (opts.SessionOrigin != "" || opts.SessionCookie != "") {
		return nil, errors.New("local document previews cannot use Host sessions or cookies")
	}
	if opts.LocalPDFViewerURL != "" {
		if runtime.GOOS != "windows" || opts.LocalPDFURL != "" || opts.LocalHTMLURL != "" || opts.LocalMediaURL != "" || strings.SplitN(raw, "#", 2)[0] != opts.LocalPDFViewerURL || !validLocalPDFViewerURL(opts.LocalPDFViewerURL) {
			return nil, errors.New("invalid PDF reader capability or unavailable WebView2 runtime")
		}
		opts.Untrusted = true
	}
	if opts.LocalMediaURL != "" {
		if runtime.GOOS != "windows" || opts.LocalPDFURL != "" || opts.LocalHTMLURL != "" || opts.LocalMediaURL != raw || !validLocalMediaURL(raw) {
			return nil, errors.New("invalid media capability or unavailable WebView2 runtime")
		}
		opts.Untrusted = true
	}
	if opts.LocalPDFURL != "" && opts.LocalHTMLURL != "" {
		return nil, errors.New("local PDF and HTML previews are mutually exclusive")
	}
	if opts.LocalHTMLURL != "" {
		if runtime.GOOS != "windows" {
			return nil, errors.New("HTML WebView preview is unavailable on this platform; use system open")
		}
		if opts.LocalHTMLURL != raw || !validLocalHTMLURL(raw) {
			return nil, errors.New("invalid local HTML view")
		}
		opts.Untrusted = true
	}
	if opts.LocalPDFURL != "" {
		if runtime.GOOS != "windows" {
			return nil, errors.New("PDF WebView preview is unavailable on this platform; use system open")
		}
		if opts.LocalPDFURL != strings.SplitN(raw, "#", 2)[0] || opts.SessionOrigin != "" {
			return nil, errors.New("invalid local PDF view")
		}
		opts.Untrusted = true
	}
	if opts.TopInset < 0 {
		opts.TopInset = 0
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Plugin Web"
	}
	var gateway *pluginslots.Gateway
	pageURL := raw
	if opts.SessionOrigin != "" {
		var err error
		gateway, err = pluginslots.NewGateway(opts.SessionOrigin, opts.SessionCookie)
		if err != nil {
			return nil, err
		}
		pageURL, err = gateway.EntryURL(raw)
		if err != nil {
			gateway.Close()
			return nil, err
		}
		opts.Untrusted = true
		gateway.SetClientModulesChanged(opts.OnChange)
		gateway.SetShortcutPreferences(opts.ShortcutPreferences)
	}
	trusted := append([]string(nil), opts.TrustedOrigins...)
	if !opts.Untrusted && len(trusted) == 0 {
		if o := urlOrigin(raw); o != "" {
			trusted = []string{o}
		}
	}

	h := &Host{localPDFViewerURL: opts.LocalPDFViewerURL, localPDFURL: opts.LocalPDFURL, localHTMLURL: opts.LocalHTMLURL, localMediaURL: opts.LocalMediaURL, onClosed: opts.OnClosed, gateway: gateway, parent: opts.Parent, topInset: opts.TopInset, url: raw, changed: opts.OnChange, loading: true}
	initialURL := pageURL
	if localPreview || opts.BrowserProfile || gateway != nil {
		// Register lifecycle and failure handlers before asynchronous load.
		initialURL = ""
	}
	child, createErr := createBrowserHostWindow(mygo.WindowOptions{
		Title:           title,
		URL:             initialURL,
		Parent:          opts.Parent,
		Frameless:       true,
		Hidden:          true,
		DisableMinimize: true,
		SkipTaskbar:     true,
		Width:           800,
		Height:          600,
		Page: mygo.PageOptions{
			BrowserProfile:    opts.BrowserProfile,
			PreloadScript:     gateway.ShortcutProviderScript(),
			BrowserPartition:  opts.BrowserPartition,
			LocalPDFViewerURL: opts.LocalPDFViewerURL,
			LocalPDFURL:       opts.LocalPDFURL,
			LocalHTMLURL:      opts.LocalHTMLURL,
			LocalMediaURL:     opts.LocalMediaURL,
			TrustedOrigins:    trusted,
			DisableBindings:   opts.Untrusted,
			DevTools:          mygo.DevToolsDisabled,
		},
	})
	if createErr != nil {
		gateway.Close()
		return nil, createErr
	}
	if child == nil || child.IsDestroyed() {
		gateway.Close()
		return nil, errors.New("webview: failed to create WebView2 window")
	}
	h.win = child
	h.installProductKeyboard()
	if !localPreview && (opts.BrowserProfile || opts.SessionOrigin != "") {
		child.Page().UseWebApplicationShortcuts(opts.Parent)
	}
	if opts.OnFocus != nil {
		child.OnFocus(opts.OnFocus)
	}
	pg := child.Page()
	pg.OnWillNavigate(func(e *mygo.NavigateEvent) {
		if (gateway != nil && !gateway.AllowsBrowserURL(e.URL)) || (opts.Untrusted && validateHTTPURL(e.URL) != nil) {
			e.PreventDefault()
			h.setError("Navigation outside this view is blocked")
			return
		}
		h.disconnectClientModulesForNavigation(e.URL)
		h.beginLoad()
	})
	pg.OnDidNavigate(func(raw string) {
		h.mu.Lock()
		h.navigationEpoch++
		h.url = h.logicalURL(raw)
		h.err = ""
		h.mu.Unlock()
		h.notify()
	})
	pg.OnDidFinishLoad(func() { h.mu.Lock(); h.loading = false; h.mu.Unlock(); h.notify() })
	pg.OnDidFailLoad(func(err *mygo.LoadError) {
		h.mu.Lock()
		stopped := h.stopped
		h.mu.Unlock()
		if !stopped {
			if localPreview {
				if opts.LocalHTMLURL != "" {
					h.setError("HTML preview failed; use system open")
				} else {
					h.setError("PDF preview failed; use system open")
				}
				_ = h.Close()
			} else if opts.BrowserProfile {
				h.setError(err.Error())
				if !pg.BrowserToolsAvailable() {
					// Environment creation failed; allow a fresh Host on retry.
					_ = h.Close()
				}
			} else if gateway != nil {
				gateway.DisconnectClientModules()
				h.setError("Plugin Host page failed to load")
			} else {
				h.setError(err.Error())
			}
		}
	})
	pg.OnRenderProcessGone(func(reason string) {
		gateway.DisconnectClientModules()
		h.setError("Page process exited: " + reason)
		if localPreview {
			_ = h.Close()
		}
	})
	pg.OnPageTitleUpdated(func(e *mygo.TitleEvent) { h.mu.Lock(); h.title = e.Title; h.mu.Unlock(); h.notify() })
	if opts.Untrusted {
		pg.SetWindowOpenHandler(func(req mygo.WindowOpenRequest) *mygo.WindowOptions {
			// Keep popups inside the same isolated page; never create a privileged child.
			if err := h.Navigate(req.URL); err != nil {
				h.setError(err.Error())
			}
			return nil
		})
	}
	h.offMove = opts.Parent.OnMove(h.SyncBounds)
	h.offResize = opts.Parent.OnResize(func() {
		h.SyncBounds()
	})
	child.OnClose(func(e *mygo.CloseEvent) {
		e.PreventDefault()
		if gateway != nil || localPreview {
			_ = h.Close()
		} else {
			h.SetVisible(false)
		}
	})
	if localPreview || gateway != nil {
		child.OnClosed(func() { _ = h.Close() })
	}
	h.SyncBounds()
	if localPreview || opts.BrowserProfile || gateway != nil {
		if err := pg.LoadURL(pageURL); err != nil {
			_ = h.Close()
			return nil, err
		}
		if !h.Alive() {
			if opts.BrowserProfile {
				return nil, errors.New("webview: Browser profile failed to start")
			}
			return nil, errors.New("webview: local document preview failed to start")
		}
	}
	created = true
	return h, nil
}

// Navigate loads an http(s) URL in the host page.
func (h *Host) Navigate(raw string) error {
	if h == nil {
		return errors.New("webview: nil host")
	}
	raw = strings.TrimSpace(raw)
	if h.localPDFViewerURL != "" && strings.SplitN(raw, "#", 2)[0] != h.localPDFViewerURL {
		return errors.New("PDF reader navigation outside this document is blocked")
	}
	if h.localPDFURL != "" && strings.SplitN(raw, "#", 2)[0] != h.localPDFURL {
		return errors.New("PDF navigation outside this document is blocked")
	}
	if h.localHTMLURL != "" && strings.SplitN(raw, "#", 2)[0] != h.localHTMLURL {
		return errors.New("HTML navigation outside this document is blocked")
	}
	if err := validateHTTPURL(raw); err != nil {
		return err
	}
	h.mu.Lock()
	win := h.win
	h.mu.Unlock()
	if win == nil || win.IsDestroyed() {
		return errors.New("webview: host closed")
	}
	pg := win.Page()
	if pg == nil {
		return errors.New("webview: no page")
	}
	pageURL := raw
	if h.gateway != nil {
		var err error
		pageURL, err = h.gateway.EntryURL(h.logicalURL(raw))
		if err != nil {
			return err
		}
	}
	h.disconnectClientModulesForNavigation(raw)
	h.beginLoad()
	if err := pg.LoadURL(pageURL); err != nil {
		if h.gateway != nil {
			err = errors.New("Plugin Host navigation failed")
		}
		h.setError(err.Error())
		if h.localPDFViewerURL != "" || h.localPDFURL != "" || h.localHTMLURL != "" {
			_ = h.Close()
		}
		return err
	}
	if (h.localPDFViewerURL != "" || h.localPDFURL != "" || h.localHTMLURL != "") && !h.Alive() {
		return errors.New("webview: local document preview closed")
	}
	h.mu.Lock()
	h.url = h.logicalURL(raw)
	h.err = ""
	h.mu.Unlock()
	h.notify()
	return nil
}

// URL returns the last navigated URL.
func (h *Host) URL() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.url
}

// SetVisible shows or hides the child window.
func (h *Host) SetVisible(visible bool) {
	if h == nil {
		return
	}
	h.mu.Lock()
	win := h.win
	h.mu.Unlock()
	if win == nil || win.IsDestroyed() {
		return
	}
	if visible {
		h.SyncBounds()
		if !win.IsVisible() {
			win.ShowInactive()
		}
	} else {
		win.Hide()
	}
}

// SetBounds moves the child in screen DIPs.
func (h *Host) SetBounds(r mygo.Rectangle) {
	if h == nil {
		return
	}
	h.mu.Lock()
	win := h.win
	h.mu.Unlock()
	if win == nil || win.IsDestroyed() {
		return
	}
	if r.Width < 1 {
		r.Width = 1
	}
	if r.Height < 1 {
		r.Height = 1
	}
	win.SetBounds(r)
}

// SyncBounds aligns the host to the parent content area below TopInset.
func (h *Host) SyncBounds() {
	if h == nil {
		return
	}
	h.mu.Lock()
	parent := h.parent
	win := h.win
	inset := h.topInset
	var seat *mygo.Rectangle
	if h.viewport != nil {
		r := *h.viewport
		seat = &r
	}
	h.mu.Unlock()
	if parent == nil || parent.IsDestroyed() || win == nil || win.IsDestroyed() {
		return
	}
	cb := parent.ContentBounds()
	win.SetBounds(viewportBounds(cb, inset, seat))
}

// Close destroys the child window and unregisters resize sync.
func (h *Host) Close() error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	off := h.offResize
	offMove := h.offMove
	h.offMove = nil
	win := h.win
	h.offResize = nil
	h.win = nil
	onClosed := h.onClosed
	h.onClosed = nil
	h.mu.Unlock()
	h.gateway.Close()
	if onClosed != nil {
		onClosed()
	}
	if offMove != nil {
		offMove()
	}
	if off != nil {
		off()
	}
	if win != nil && !win.IsDestroyed() {
		win.Destroy()
	}
	return nil
}

// Alive reports whether the host window still exists.
func (h *Host) Alive() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.win != nil && !h.win.IsDestroyed()
}

func validateHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Hostname() == "" || u.User != nil {
			return errors.New("webview: invalid host or URL credentials")
		}
		return nil
	default:
		return errors.New("webview: only http(s) URLs are allowed")
	}
}

// validLocalHTMLURL mirrors the framework's narrow capability shape before
// NewWindow, whose invalid-option contract is to panic. Never accept an origin,
// directory, query, escaped path, credentials or an arbitrary loopback URL.
func validLocalHTMLURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Opaque != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || u.String() != raw {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || u.Host != "127.0.0.1:"+strconv.Itoa(port) {
		return false
	}
	parts := strings.Split(u.Path, "/")
	return len(parts) == 3 && parts[0] == "" && len(parts[1]) == 64 && strings.Trim(parts[1], "0123456789abcdefABCDEF") == "" && parts[2] == "document.html"
}

func validLocalMediaURL(raw string) bool {
	if !strings.HasSuffix(raw, "/player.html") {
		return false
	}
	return validLocalHTMLURL(strings.TrimSuffix(raw, "player.html") + "document.html")
}

func urlOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// State reads native navigation state, including link clicks and redirects.
type State struct {
	URL, Title, Error            string
	Loading, CanBack, CanForward bool
}

func (h *Host) State() State {
	if h == nil {
		return State{}
	}
	h.mu.Lock()
	s := State{URL: h.url, Title: h.title, Error: h.err, Loading: h.loading}
	w := h.win
	stopped := h.stopped
	h.mu.Unlock()
	if w != nil && !w.IsDestroyed() {
		p := w.Page()
		s.Loading = s.Loading || (!stopped && p.IsLoading())
		s.CanBack = p.CanGoBack()
		s.CanForward = p.CanGoForward()
		if u := p.URL(); u != "" {
			s.URL = h.logicalURL(u)
		}
	}
	return s
}
func (h *Host) notify() {
	if h.changed != nil {
		h.changed()
	}
}
func (h *Host) setError(err string) {
	h.mu.Lock()
	h.err = err
	if err != "" {
		h.loading = false
	}
	h.mu.Unlock()
	h.notify()
}
func (h *Host) pageAction(fn func(*mygo.Page)) {
	if h == nil {
		return
	}
	h.mu.Lock()
	w := h.win
	h.mu.Unlock()
	if w != nil && !w.IsDestroyed() {
		fn(w.Page())
		h.notify()
	}
}
func (h *Host) Back()    { h.pageAction(func(p *mygo.Page) { p.GoBack() }) }
func (h *Host) Forward() { h.pageAction(func(p *mygo.Page) { p.GoForward() }) }
func (h *Host) Reload()  { h.setError(""); h.pageAction(func(p *mygo.Page) { p.Reload() }) }
func (h *Host) Stop() {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.stopped = true
	h.loading = false
	h.err = ""
	h.mu.Unlock()
	h.pageAction(func(p *mygo.Page) { p.Stop() })
}

func (h *Host) beginLoad() {
	h.mu.Lock()
	h.navigationEpoch++
	h.loading = true
	h.stopped = false
	h.err = ""
	h.mu.Unlock()
	h.notify()
}

// MatchesSession prevents reusing a page after Host restart or credential change.
func (h *Host) MatchesSession(origin, cookie string) bool {
	if h == nil {
		return false
	}
	if h.gateway == nil {
		return origin == ""
	}
	return h.gateway.Matches(origin, cookie)
}
func (h *Host) HasSession() bool { return h != nil && h.gateway != nil }
func (h *Host) logicalURL(raw string) string {
	if h.gateway != nil {
		return h.gateway.LogicalURL(raw)
	}
	return raw
}

// IsHostURL requires an exact scheme, host and port, with no URL credentials.
func IsHostURL(raw, origin string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return false
	}
	return origin != "" && urlOrigin(raw) == strings.TrimRight(origin, "/")
}
