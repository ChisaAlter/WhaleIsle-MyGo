package webview

import (
	"errors"
	"net/url"
	"strings"
	"sync"

	"github.com/egoist/mygo"
)

// HostOptions configure a borderless WebView2 child window.
type HostOptions struct {
	// Untrusted disables all native bindings and app events for browsing.
	Untrusted bool
	OnChange  func()
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
	mu        sync.Mutex
	parent    *mygo.Window
	win       *mygo.Window
	topInset  int
	url       string
	offResize func()
	offMove   func()
	loading   bool
	stopped   bool
	err       string
	title     string
	changed   func()
}

// NewHost creates a hidden frameless child WebView2 window. Call
// SyncBounds then SetVisible(true) after the parent has a real size.
func NewHost(opts HostOptions) (*Host, error) {
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
	if opts.TopInset < 0 {
		opts.TopInset = 0
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Plugin Web"
	}
	trusted := append([]string(nil), opts.TrustedOrigins...)
	if !opts.Untrusted && len(trusted) == 0 {
		if o := urlOrigin(raw); o != "" {
			trusted = []string{o}
		}
	}

	h := &Host{parent: opts.Parent, topInset: opts.TopInset, url: raw, changed: opts.OnChange, loading: true}
	child := mygo.NewWindow(mygo.WindowOptions{
		Title:           title,
		URL:             raw,
		Parent:          opts.Parent,
		Frameless:       true,
		Hidden:          true,
		DisableMinimize: true,
		SkipTaskbar:     true,
		Width:           800,
		Height:          600,
		Page: mygo.PageOptions{
			TrustedOrigins:  trusted,
			DisableBindings: opts.Untrusted,
			DevTools:        mygo.DevToolsDisabled,
		},
	})
	if child == nil || child.IsDestroyed() {
		return nil, errors.New("webview: failed to create WebView2 window")
	}
	h.win = child
	pg := child.Page()
	pg.OnWillNavigate(func(e *mygo.NavigateEvent) {
		if opts.Untrusted && validateHTTPURL(e.URL) != nil {
			e.PreventDefault()
			h.setError("Only HTTP(S) navigation is allowed")
			return
		}
		h.beginLoad()
	})
	pg.OnDidNavigate(func(raw string) { h.mu.Lock(); h.url = raw; h.err = ""; h.mu.Unlock(); h.notify() })
	pg.OnDidFinishLoad(func() { h.mu.Lock(); h.loading = false; h.mu.Unlock(); h.notify() })
	pg.OnDidFailLoad(func(err *mygo.LoadError) {
		h.mu.Lock()
		stopped := h.stopped
		h.mu.Unlock()
		if !stopped {
			h.setError(err.Error())
		}
	})
	pg.OnRenderProcessGone(func(reason string) { h.setError("Page process exited: " + reason) })
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
		h.SetVisible(false)
	})
	h.SyncBounds()
	return h, nil
}

// Navigate loads an http(s) URL in the host page.
func (h *Host) Navigate(raw string) error {
	if h == nil {
		return errors.New("webview: nil host")
	}
	raw = strings.TrimSpace(raw)
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
	h.beginLoad()
	if err := pg.LoadURL(raw); err != nil {
		h.setError(err.Error())
		return err
	}
	h.mu.Lock()
	h.url = raw
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
	h.mu.Unlock()
	if parent == nil || parent.IsDestroyed() || win == nil || win.IsDestroyed() {
		return
	}
	cb := parent.ContentBounds()
	y := cb.Y + inset
	height := cb.Height - inset
	if height < 1 {
		height = 1
	}
	win.SetBounds(mygo.Rectangle{X: cb.X, Y: y, Width: cb.Width, Height: height})
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
	h.mu.Unlock()
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
			s.URL = u
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
	h.loading = true
	h.stopped = false
	h.err = ""
	h.mu.Unlock()
	h.notify()
}
