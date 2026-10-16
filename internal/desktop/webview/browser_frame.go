package webview

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"math"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/browserrecording"
)

// CaptureBrowserFrame is a bounded, explicitly requested PiP/recording consumer of one
// existing browser page. It captures that WebView's pixels, never the desktop.
// Unlike an interactive picker it may keep the same source when its tab hides.
// A navigation changes the epoch and ends this capability rather than silently
// showing a different document, a replacement Host, a PDF or a plugin session.
func (h *Host) CaptureBrowserFrame(ctx context.Context, epoch uint64) ([]byte, error) {
	return h.captureBrowserFrame(ctx, epoch, false)
}

// CaptureBrowserRecordingFrame preserves the original recording viewport range.
// Only explicit recording uses this larger bound; PiP/picker defaults are unchanged.
func (h *Host) CaptureBrowserRecordingFrame(ctx context.Context, epoch uint64) ([]byte, error) {
	return h.captureBrowserFrame(ctx, epoch, true)
}

func (h *Host) captureBrowserFrame(ctx context.Context, epoch uint64, recording bool) ([]byte, error) {
	if h == nil {
		return nil, errors.New("browser: no page capture source")
	}
	h.mu.Lock()
	w, raw := h.win, h.url
	restricted := h.localPDFViewerURL != "" || h.localPDFURL != "" || h.localHTMLURL != "" || h.localMediaURL != "" || h.gateway != nil
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
	maxDimension, maxPixels, maxBytes := 2048, int64(4*1024*1024), 8*1024*1024
	scale := 1.0
	if recording {
		// Original ui-preview viewport.ts permits 3840 per axis and 4K area.
		maxDimension, maxPixels, maxBytes = 3840, 3840*2160, browserrecording.MaxFrameBytes
		scale = mygo.Screen.DisplayNearestPoint(mygo.Point{X: bounds.X + bounds.Width/2, Y: bounds.Y + bounds.Height/2}).ScaleFactor
		if math.IsNaN(scale) || math.IsInf(scale, 0) || scale < 1 {
			scale = 1
		}
		if float64(bounds.Width)*float64(bounds.Height)*scale*scale > float64(browserrecording.MaxFramePixels) {
			return nil, errors.New("browser: native recording pixels exceed the bounded frame budget")
		}
	}
	if bounds.Width < 1 || bounds.Height < 1 || bounds.Width > maxDimension || bounds.Height > maxDimension || int64(bounds.Width)*int64(bounds.Height) > maxPixels {
		return nil, errors.New("browser: source viewport exceeds the page capture range")
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
		if len(captured.data) == 0 || len(captured.data) > maxBytes {
			return nil, errors.New("browser: page capture PNG exceeds the frame byte budget")
		}
		config, err := png.DecodeConfig(bytes.NewReader(captured.data))
		if err != nil {
			return nil, err
		}
		if recording {
			// Source dimensions come from native window/display APIs, never page JS.
			// A resize may finish during capture. Either native footprint is valid;
			// the encoder letterboxes it into the first frame's fixed dimensions.
			now := w.ContentBounds()
			matchesBefore := recordingFrameDimensions(config.Width, config.Height, bounds.Width, bounds.Height, scale)
			matchesAfter := now.Width > 0 && now.Height > 0 && now.Width <= 3840 && now.Height <= 3840 && int64(now.Width)*int64(now.Height) <= 3840*2160 && recordingFrameDimensions(config.Width, config.Height, now.Width, now.Height, scale)
			if !matchesBefore && !matchesAfter {
				return nil, errors.New("browser: recording PNG dimensions differ from the native viewport")
			}
			return captured.data, nil
		}
		if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || int64(config.Width)*int64(config.Height) > 4*1024*1024 {
			return nil, errors.New("browser: page capture frame exceeds 4 megapixels; reduce the source viewport")
		}
		return captured.data, nil
	}
}

// Native backends may produce logical-size or display-scale PNGs. Allow one
// pixel of native rounding, but never arbitrary dimensions from PNG metadata.
func recordingFrameDimensions(width, height, viewportWidth, viewportHeight int, scale float64) bool {
	if width < 1 || height < 1 || int64(width)*int64(height) > browserrecording.MaxFramePixels {
		return false
	}
	matches := func(factor float64) bool {
		return math.Abs(float64(width)-float64(viewportWidth)*factor) <= 1 && math.Abs(float64(height)-float64(viewportHeight)*factor) <= 1
	}
	return matches(1) || matches(scale)
}
