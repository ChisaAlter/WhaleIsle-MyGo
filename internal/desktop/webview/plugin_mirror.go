package webview

import (
	"errors"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginslots"
)

// PluginMirrorSource returns only the original binding-free page of this exact
// registered capability. It keeps the HWND hidden and never creates a second
// browser, grants bindings or replaces navigation/permission/download policy.
// The caller must sample/capture asynchronously, never from native paint.
func (h *Host) PluginMirrorSource(capability *pluginslots.Gateway, width, height int) (*mygo.Window, error) {
	if h == nil || capability == nil || width < 1 || width > 2048 || height < 1 || height > 1536 {
		return nil, errors.New("plugin mirror: bounded source required")
	}
	return h.pluginSource(capability, width, height, true)
}

// PluginRichSource retains the same binding-free page for live rich controls.
// It hides before resizing; only an acknowledged original-owner layout may be
// presented with PresentPlugin. It grants no new browser or helper capability.
func (h *Host) PluginRichSource(capability *pluginslots.Gateway, width, height int) (*mygo.Window, error) {
	if h == nil || capability == nil || width < 1 || width > 8192 || height < 1 || height > 8192 {
		return nil, errors.New("plugin rich content: bounded source required")
	}
	return h.pluginSource(capability, width, height, false)
}

func (h *Host) pluginSource(capability *pluginslots.Gateway, width, height int, hide bool) (*mygo.Window, error) {
	registered, _ := pluginSurfaceCapabilities.Load(h)
	if registered != capability {
		return nil, errors.New("plugin source: exact registered capability required")
	}
	var result *mygo.Window
	var failure = errors.New("plugin mirror: original source unavailable")
	mygo.RunOnMain(func() {
		h.mu.Lock()
		w, parent := h.win, h.parent
		registered, _ := pluginSurfaceCapabilities.Load(h)
		allowed := registered == capability && h.gateway == nil && h.allowNavigation != nil && capability.AllowsBrowserURL(h.initialBaseURL) && h.allowNavigation(h.initialBaseURL) && h.localPDFURL == "" && h.localPDFViewerURL == "" && h.localHTMLURL == "" && h.localMediaURL == ""
		if allowed {
			h.mediaSeat = true
		}
		h.mu.Unlock()
		if !allowed || w == nil || parent == nil || w.IsDestroyed() || parent.IsDestroyed() || !parent.IsVisible() || parent.IsMinimized() {
			return
		}
		bounds := w.ContentBounds()
		if hide || !w.IsVisible() || bounds.Width != width || bounds.Height != height {
			w.Hide()
		}
		if bounds.Width != width || bounds.Height != height {
			w.SetContentSize(width, height)
		}
		result, failure = w, nil
	})
	return result, failure
}
