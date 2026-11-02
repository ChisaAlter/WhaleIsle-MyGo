package pluginui

import (
	"context"
	"reflect"
	"time"
)

// probeMirror only reuses a frame when both source geometry and its text-input
// pixels agree with the captured sample. Acknowledged input marks the source
// dirty independently of metadata/textContent (input.value is not textContent).
func (o *Owner) probeMirror(frame *MirrorFrame) bool {
	if frame == nil || frame.source == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	geometry, err := mirrorEval[mirrorGeometry](ctx, frame.source.Page(), "window.__whaleInlineMirror.snapshot()")
	if err != nil || !reflect.DeepEqual(geometry, frame.geometry) {
		return false
	}
	focus, err := mirrorEval[*mirrorTextFocus](ctx, frame.source.Page(), "window.__whaleInlineMirror.textFocus()")
	if err != nil {
		return false
	}
	sampled := time.Now()
	o.mu.Lock()
	dirty := o.mirror == nil || o.mirror.pixelDirty
	if o.mirrorCurrentLocked(frame) && focus.valid() {
		state := o.mirrorTextLocked()
		client := state.clients[focus.Handle]
		if client != nil && client.pending == 0 {
			client.applyTextSampleLocked(focus, sampled)
			client.source, client.page = frame.source, frame.page
		}
	}
	o.mu.Unlock()
	return !dirty && !geometry.Animated && reflect.DeepEqual(focus, frame.textFocus)
}
