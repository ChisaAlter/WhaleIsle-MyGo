package pluginui

import (
	"context"
	"math"
	"time"

	"github.com/coder/websocket/wsjson"
)

type mirrorNativeAnchor struct {
	known    bool
	tail     SessionTurnTail
	fragment bool
	u, v     float64
}

// RequestMirrorTail scrolls only to one existing original retained occurrence.
// It does not move/reparent nodes or make another React owner. The worker
// repeats authority checks before the fixed metadata message is sent.
func (o *Owner) RequestMirrorTail(page, revision uint64, subject Subject, tail SessionTurnTail, handle string) error {
	return o.requestMirrorAnchor(page, revision, subject, tail, handle, mirrorNativeAnchor{})
}
func (o *Owner) requestMirrorAnchor(page, revision uint64, subject Subject, tail SessionTurnTail, handle string, anchor mirrorNativeAnchor) error {
	if o == nil {
		return errSessionUnavailable
	}
	o.mu.Lock()
	valid := o.mirror != nil && !o.closed && !o.masked && !o.incompatible && o.state.Connected && o.state.Page == page && o.state.Revision == revision && o.state.Subject != nil && o.state.Subject.equal(subject) && o.selected.equal(subject) && !o.mirror.inputBusy
	found := false
	for _, row := range o.state.Outlets {
		if row.Slot == "conversation.chat.turnTail" && row.Handle == handle && row.HasTurn && row.HasTurnTail && row.Available && row.Turn == tail.Turn && row.TurnTailSeq == tail.Seq {
			found = true
			break
		}
	}
	target := o.page
	state := o.mirror
	valid = valid && target != nil && target.id == page
	if valid && found {
		state.inputBusy = true
		o.invalidateMirrorLocked()
	}
	o.mu.Unlock()
	if !valid || !found || target == nil || target.id != page {
		return errSessionUnavailable
	}
	go func() {
		defer func() {
			o.mu.Lock()
			if o.mirror == state {
				state.inputBusy = false
			}
			o.mu.Unlock()
			o.notify()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if o.awaitMirrorCapture(ctx, state) != nil {
			return
		}
		o.mu.Lock()
		current := o.mirror == state && state.gesture == nil && state.inputBusy && !o.closed && o.page == target && o.state.Page == page && o.state.Revision == revision && o.selected.equal(subject)
		o.mu.Unlock()
		if !current {
			return
		}
		if err := wsjson.Write(ctx, target.socket, map[string]any{"type": "inlineTail", "revision": revision, "subject": subject, "handle": handle, "turnTail": tail, "fragment": anchor.fragment, "u": anchor.u, "v": anchor.v}); err != nil {
			o.disconnect(target)
		}
	}()
	return nil
}

// NoteMirrorNativeTail is paint-only metadata. It neither invalidates a source
// frame nor performs a browser operation. Native frame build consumes this one
// canonical anchor before requesting its next asynchronous source capture.
func (o *Owner) NoteMirrorNativeTail(tail SessionTurnTail) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.mirror != nil && !o.closed {
		o.mirror.pendingAnchor = mirrorNativeAnchor{known: true, tail: tail}
	}
}
func (o *Owner) consumeMirrorNativeTail() {
	o.mu.Lock()
	m := o.mirror
	if m == nil || !m.pendingAnchor.known || m.inputBusy {
		o.mu.Unlock()
		return
	}
	next := m.pendingAnchor
	m.pendingAnchor = mirrorNativeAnchor{}
	if m.anchor.known && m.anchor.tail == next.tail && m.anchor.fragment == next.fragment && (!next.fragment || math.Abs(m.anchor.u-next.u) < .5 && math.Abs(m.anchor.v-next.v) < .5) {
		o.mu.Unlock()
		return
	}
	m.anchor = next
	frame := m.frame
	snapshot := o.state
	o.mu.Unlock()
	if frame == nil || snapshot.Subject == nil {
		return
	}
	for _, row := range frame.Occurrences {
		if row.Turn == next.tail.Turn && row.Seq == next.tail.Seq && row.Visible && !next.fragment {
			return
		}
	}
	for _, row := range snapshot.Outlets {
		if row.Slot == "conversation.chat.turnTail" && row.Available && row.HasTurnTail && row.HasTurn && row.Turn == next.tail.Turn && row.TurnTailSeq == next.tail.Seq {
			_ = o.requestMirrorAnchor(snapshot.Page, snapshot.Revision, *snapshot.Subject, next.tail, row.Handle, next)
			return
		}
	}
}
