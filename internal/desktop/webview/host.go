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
	if len(trusted) == 0 {
		if o := urlOrigin(raw); o != "" {
			trusted = []string{o}
		}
	}

	h := &Host{parent: opts.Parent, topInset: opts.TopInset, url: raw}
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
			TrustedOrigins: trusted,
			DevTools:       mygo.DevToolsDisabled,
		},
	})
	if child == nil || child.IsDestroyed() {
		return nil, errors.New("webview: failed to create WebView2 window")
	}
	h.win = child
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
	h.url = raw
	h.mu.Unlock()
	if win == nil || win.IsDestroyed() {
		return errors.New("webview: host closed")
	}
	pg := win.Page()
	if pg == nil {
		return errors.New("webview: no page")
	}
	return pg.LoadURL(raw)
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
		win.Show()
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
	win := h.win
	h.offResize = nil
	h.win = nil
	h.mu.Unlock()
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
