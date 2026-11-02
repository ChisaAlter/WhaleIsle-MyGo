package webview

import (
	"errors"
	"slices"
	"sync"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginslots"
)

type mediaPlacement struct {
	bounds mygo.Rectangle
	holes  []mygo.Rectangle
}

func clippedSurfaceHoles(bounds, visible mygo.Rectangle, occlusions []mygo.Rectangle) []mygo.Rectangle {
	left, top := visible.X-bounds.X, visible.Y-bounds.Y
	right, bottom := left+visible.Width, top+visible.Height
	holes := make([]mygo.Rectangle, 0, len(occlusions)+4)
	for _, r := range []mygo.Rectangle{{Width: bounds.Width, Height: top}, {Y: bottom, Width: bounds.Width, Height: bounds.Height - bottom}, {Y: top, Width: left, Height: visible.Height}, {X: right, Y: top, Width: bounds.Width - right, Height: visible.Height}} {
		if r.Width > 0 && r.Height > 0 {
			holes = append(holes, r)
		}
	}
	for _, r := range occlusions {
		r = intersectMediaRect(r, bounds)
		if r.Width > 0 && r.Height > 0 {
			r.X -= bounds.X
			r.Y -= bounds.Y
			holes = append(holes, r)
		}
	}
	return holes
}

func intersectMediaRect(a, b mygo.Rectangle) mygo.Rectangle {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	right, bottom := min(a.X+a.Width, b.X+b.Width), min(a.Y+a.Height, b.Y+b.Height)
	return mygo.Rectangle{X: x, Y: y, Width: max(0, right-x), Height: max(0, bottom-y)}
}

// PresentMedia is a narrow paint-owned alternative to whole-panel SetViewport.
// Bounds/clip/occlusions are trusted parent-content DIPs, never renderer data.
// Unlike SetViewport it preserves the unclipped player's size when an edge is
// outside the scroll viewport. All invisible pixels AND input are excluded.
// A platform unable to confirm clipping hides the page and reports failure.
// Ordinary Browser and document Hosts never opt into this path.
func (h *Host) PresentMedia(bounds, clip mygo.Rectangle, occlusions []mygo.Rectangle) (bool, error) {
	return h.presentClipped(bounds, clip, occlusions, nil)
}

var pluginSurfaceCapabilities sync.Map // *Host -> *pluginslots.Gateway, native caller owned

// NewPluginSurface creates only a binding-free exact-gateway page. The owner
// retains and closes its capability; it never receives application bindings.
func NewPluginSurface(opts HostOptions, capability *pluginslots.Gateway) (*Host, error) {
	if capability == nil || !capability.AllowsBrowserURL(opts.URL) || opts.SessionOrigin != "" || opts.SessionCookie != "" {
		return nil, errors.New("plugin UI: exact browser capability required")
	}
	opts.Untrusted = true
	opts.TrustedOrigins = nil
	opts.AllowNavigation = capability.AllowsBrowserURL
	closed := opts.OnClosed
	var host *Host
	opts.OnClosed = func() {
		if host != nil {
			pluginSurfaceCapabilities.Delete(host)
		}
		if closed != nil {
			closed()
		}
	}
	var err error
	host, err = NewHost(opts)
	if err == nil {
		// A fixed native seat has no resize frame. Keeping the default invisible
		// thick frame would shrink the DOM viewport inside these exact bounds.
		host.win.SetResizable(false)
		pluginSurfaceCapabilities.Store(host, capability)
	}
	return host, err
}

// PresentPlugin preserves a contribution's full viewport while native scrolling
// clips its pixels and input. The exact-Host capability must authorize this
// binding-free Host; a caller cannot use the path for ordinary browser pages.
func (h *Host) PresentPlugin(bounds, clip mygo.Rectangle, occlusions []mygo.Rectangle, capability *pluginslots.Gateway) (bool, error) {
	if h == nil || capability == nil {
		return false, errors.New("plugin UI: Host capability required")
	}
	return h.presentClipped(bounds, clip, occlusions, capability)
}

func (h *Host) presentClipped(bounds, clip mygo.Rectangle, occlusions []mygo.Rectangle, capability *pluginslots.Gateway) (bool, error) {
	if h == nil {
		return false, errors.New("media: native browser is unavailable")
	}
	shown, result := false, errors.New("media: application UI loop is unavailable")
	mygo.RunOnMain(func() {
		h.mu.Lock()
		w, parent := h.win, h.parent
		private := h.transient && h.gateway == nil && h.localPDFURL == "" && h.localPDFViewerURL == "" && h.localHTMLURL == "" && h.localMediaURL == ""
		if capability != nil {
			// This is an application-owned authenticated gateway, not a plugin URL
			// claim. Construction guarantees disabled bindings; require its pinned
			// browser origin and navigation guard for every subsequent paint too.
			registered, _ := pluginSurfaceCapabilities.Load(h)
			private = registered == capability && h.gateway == nil && h.allowNavigation != nil && capability.AllowsBrowserURL(h.initialBaseURL) && h.allowNavigation(h.initialBaseURL) && h.localPDFURL == "" && h.localPDFViewerURL == "" && h.localHTMLURL == "" && h.localMediaURL == ""
		}
		if private {
			h.mediaSeat = true
		}
		previous := h.mediaPlacement
		h.mu.Unlock()
		if !private || w == nil || parent == nil || w.IsDestroyed() || parent.IsDestroyed() {
			result = errors.New("media: a live private native browser is required")
			return
		}
		origin := parent.ContentBounds()
		visible := intersectMediaRect(intersectMediaRect(bounds, clip), mygo.Rectangle{Width: origin.Width, Height: origin.Height})
		if !parent.IsVisible() || parent.IsMinimized() || bounds.Width < 1 || bounds.Height < 1 || visible.Width < 1 || visible.Height < 1 {
			w.Hide()
			result = nil
			return
		}
		if bounds.Width > 8192 || bounds.Height > 8192 || bounds.X < -1e7 || bounds.X > 1e7 || bounds.Y < -1e7 || bounds.Y > 1e7 {
			w.Hide()
			result = errors.New("media: native paint bounds exceed the surface range")
			return
		}
		holes := clippedSurfaceHoles(bounds, visible, occlusions)
		screen := bounds
		screen.X += origin.X
		screen.Y += origin.Y
		if previous == nil || previous.bounds != screen || !slices.Equal(previous.holes, holes) {
			// Never move/show a surface with a stale region from its last seat.
			w.Hide()
			w.SetBounds(screen)
			if err := clipMediaNativeWindow(w, bounds.Width, bounds.Height, holes); err != nil {
				result = err
				return
			}
			h.mu.Lock()
			h.mediaPlacement = &mediaPlacement{bounds: screen, holes: append([]mygo.Rectangle(nil), holes...)}
			h.mu.Unlock()
		}
		if !w.IsVisible() {
			w.ShowInactive()
		}
		shown, result = true, nil
	})
	return shown, result
}
