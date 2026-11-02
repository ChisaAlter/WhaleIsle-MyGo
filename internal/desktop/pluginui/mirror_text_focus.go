package pluginui

import (
	"github.com/egoist/mygo"
	"time"
)

// bindMirrorOriginalFocus applies only a read-only context from the original
// trusted click. Keeping it independent of a screenshot lets native IME start
// immediately while the next shared bitmap is still being captured.
func (o *Owner) bindMirrorOriginalFocus(frame *MirrorFrame, focus *mirrorTextFocus) {
	if !focus.valid() || frame == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.mirrorLayoutCurrentLocked(frame) || focus.DocumentKey != frame.geometry.DocumentKey || !focus.Subject.equal(frame.subject) {
		return
	}
	state := o.mirrorTextLocked()
	client := state.clients[focus.Handle]
	if client == nil {
		client = &MirrorTextClient{owner: o, handle: focus.Handle}
		state.clients[focus.Handle] = client
	}
	client.applyFocusLocked(focus)
	client.updated = time.Now()
	client.source, client.page = frame.source, frame.page
}

// MirrorTextObservation exposes cached state for diagnostics without querying,
// focusing or changing the original browser document.
func (o *Owner) MirrorTextObservation() any {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mirror == nil || o.mirror.text == nil {
		return nil
	}
	rows := map[string]any{}
	for id, c := range o.mirror.text.clients {
		rows[id] = map[string]any{"focus": c.focus, "pending": c.pending, "selection": c.selection, "marked": c.marked, "value": c.value}
	}
	return map[string]any{"clients": rows, "busy": o.mirror.text.busy, "queued": len(o.mirror.text.queue), "closed": o.mirror.text.closed}
}

func (o *Owner) cancelMirrorText(source *mygo.Window, focus *mirrorTextFocus) {
	if source == nil || focus == nil || source.IsDestroyed() {
		return
	}
	// Cleanup implementation below shares the original document/focus fence. It
	// never replays key releases or commits text into a newly focused element.
	o.cancelMirrorTextAt(source, focus)
}
