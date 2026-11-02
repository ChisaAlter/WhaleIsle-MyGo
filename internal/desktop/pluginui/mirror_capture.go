package pluginui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"reflect"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/webview"
)

func (o *Owner) captureMirror(h *webview.Host, capability *pluginslots.Gateway, width, height int, subject Subject, page, revision, generation uint64) (*MirrorFrame, error) {
	if h == nil || capability == nil {
		return nil, errSessionUnavailable
	}
	source, err := h.PluginMirrorSource(capability, width, height)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	before, err := mirrorEval[mirrorGeometry](ctx, source.Page(), "window.__whaleInlineMirror?.snapshot()")
	if err != nil {
		return nil, err
	}
	if !validMirrorGeometry(before, subject) {
		return nil, errSessionUnavailable
	}
	// CapturePreview has no native cancellation. The single inflight reservation
	// is held until real completion even when source identity is invalidated.
	pixels, err := source.CapturePage()
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("plugin mirror: native CapturePage elapsed: %w", ctx.Err())
	}
	after, err := mirrorEval[mirrorGeometry](ctx, source.Page(), "window.__whaleInlineMirror?.snapshot()")
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(before, after) {
		return nil, errSessionUnavailable
	}
	if len(pixels) == 0 || len(pixels) > 8<<20 {
		return nil, errors.New("plugin mirror: bounded image unavailable")
	}
	config, err := png.DecodeConfig(bytes.NewReader(pixels))
	if err != nil {
		return nil, err
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || int64(config.Width)*int64(config.Height) > 4<<20 {
		return nil, errors.New("plugin mirror: image exceeds source budget")
	}
	bitmap, err := ui.DecodeBitmap(pixels)
	if err != nil {
		return nil, err
	}
	o.mu.Lock()
	valid := !o.closed && !o.masked && o.mirror != nil && o.mirror.generation == generation && o.state.Page == page && o.state.Revision == revision && o.selected.equal(subject)
	o.mu.Unlock()
	if !valid {
		return nil, errSessionUnavailable
	}
	focus, focusErr := mirrorEval[*mirrorTextFocus](ctx, source.Page(), "window.__whaleInlineMirror.textFocus()")
	if focusErr != nil {
		return nil, fmt.Errorf("plugin mirror: text focus query: %w", focusErr)
	}
	return &MirrorFrame{textAt: time.Now(), textFocus: focus, Bitmap: bitmap, PNGWidth: config.Width, PNGHeight: config.Height, Occurrences: append([]MirrorOccurrence(nil), after.Owners...), subject: subject, page: page, revision: revision, generation: generation, geometry: after, source: source}, nil
}
