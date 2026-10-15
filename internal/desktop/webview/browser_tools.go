package webview

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/egoist/mygo"
)

// BrowserPageVersion identifies navigation within a Host. Combined with Host
// pointer identity it prevents a late toolbar result reaching a replacement tab.
func (h *Host) BrowserPageVersion() uint64 {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.navigationEpoch
}
func (h *Host) BrowserZoom() float64 {
	if h == nil {
		return 1
	}
	h.mu.Lock()
	w := h.win
	h.mu.Unlock()
	if w == nil || w.IsDestroyed() || w.Page() == nil {
		return 1
	}
	z := w.Page().ZoomFactor()
	if z <= 0 || math.IsNaN(z) || math.IsInf(z, 0) {
		return 1
	}
	return z
}
func (h *Host) SetBrowserZoom(zoom float64) error {
	if math.IsNaN(zoom) || math.IsInf(zoom, 0) || zoom < 0.25 || zoom > 5 {
		return errors.New("browser: zoom must be between 25% and 500%")
	}
	w, _, _, err := h.browserPage()
	if err != nil {
		return err
	}
	w.Page().SetZoomFactor(zoom)
	h.notify()
	return nil
}
func (h *Host) BrowserToolsAvailable() bool {
	w, _, _, err := h.browserPage()
	return err == nil && w.Page().BrowserToolsAvailable()
}
func (h *Host) HardReloadBrowser() error {
	w, _, _, err := h.browserPage()
	if err != nil {
		return err
	}
	h.beginLoad()
	w.Page().ReloadIgnoringCache()
	return nil
}
func (h *Host) SetBrowserColorScheme(ctx context.Context, scheme mygo.BrowserColorScheme) error {
	return h.browserTool(ctx, func(ctx context.Context, p *mygo.Page) error { return p.SetBrowserColorScheme(ctx, scheme) })
}

// The caller must confirm the shared-profile HTTP cache scope before this call.
func (h *Host) ClearBrowserHTTPCache(ctx context.Context) error {
	return h.browserTool(ctx, func(ctx context.Context, p *mygo.Page) error { return p.ClearBrowserHTTPCache(ctx) })
}
func (h *Host) browserTool(ctx context.Context, run func(context.Context, *mygo.Page) error) error {
	w, epoch, _, err := h.browserPage()
	if err != nil {
		return err
	}
	ctx, stop := pageOperationContext(ctx, w, 15*time.Second)
	defer stop()
	if ctx.Err() != nil || !h.sameBrowserPage(w, epoch) {
		return context.Canceled
	}
	err = run(ctx, w.Page())
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !h.sameBrowserPage(w, epoch) {
		return context.Canceled
	}
	return err
}
