package webview

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"time"
)

// CaptureBrowserFrame is a bounded, explicitly requested PiP/recording consumer of one
// existing browser page. It captures that WebView's pixels, never the desktop.
// Unlike an interactive picker it may keep the same source when its tab hides.
// A navigation changes the epoch and ends this capability rather than silently
// showing a different document, a replacement Host, a PDF or a plugin session.
func (h *Host) CaptureBrowserFrame(ctx context.Context, epoch uint64) ([]byte, error) {
	if h == nil {
		return nil, errors.New("browser: no page capture source")
	}
	h.mu.Lock()
	w, raw := h.win, h.url
	restricted := h.localPDFURL != "" || h.localHTMLURL != "" || h.localMediaURL != "" || h.gateway != nil
	current := h.navigationEpoch == epoch && !h.loading && h.err == ""
	h.mu.Unlock()
	if restricted || !current || w == nil || w.IsDestroyed() || w.Page() == nil {
		return nil, context.Canceled
	}
	if err := validateHTTPURL(raw); err != nil {
		return nil, err
	}
	ctx, stop := pageOperationContext(ctx, w, 10*time.Second)
	defer stop()
	if ctx.Err() != nil || !h.sameBrowserPage(w, epoch) {
		return nil, context.Canceled
	}
	bounds := w.ContentBounds()
	if bounds.Width < 1 || bounds.Height < 1 || bounds.Width > 2048 || bounds.Height > 2048 || int64(bounds.Width)*int64(bounds.Height) > 4*1024*1024 {
		return nil, errors.New("browser: resize the source viewport below 2048 × 2048 for page capture")
	}
	h.mu.Lock()
	if h.capturePending {
		h.mu.Unlock()
		return nil, errors.New("browser: another native page capture is still finishing")
	}
	h.capturePending = true
	h.mu.Unlock()
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := w.CapturePage()
		h.mu.Lock()
		h.capturePending = false
		h.mu.Unlock()
		done <- result{data, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case captured := <-done:
		if ctx.Err() != nil || !h.sameBrowserPage(w, epoch) {
			return nil, context.Canceled
		}
		if captured.err != nil {
			return nil, captured.err
		}
		if len(captured.data) == 0 || len(captured.data) > 8*1024*1024 {
			return nil, errors.New("browser: page capture frame exceeds the 8 MiB limit")
		}
		config, err := png.DecodeConfig(bytes.NewReader(captured.data))
		if err != nil {
			return nil, err
		}
		if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || int64(config.Width)*int64(config.Height) > 4*1024*1024 {
			return nil, errors.New("browser: page capture frame exceeds 4 megapixels; reduce the source viewport")
		}
		return captured.data, nil
	}
}
