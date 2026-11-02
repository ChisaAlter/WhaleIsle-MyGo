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
		w.Hide()
		bounds := w.ContentBounds()
		if bounds.Width != width || bounds.Height != height {
			w.SetContentSize(width, height)
		}
		result, failure = w, nil
	})
	return result, failure
}
