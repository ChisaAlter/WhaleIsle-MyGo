package pluginui

import (
	"context"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMirrorFastReleaseIsQueuedWhileNativeDownBusy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := &mirrorGesture{row: MirrorOccurrence{Handle: "original-A"}, down: mirrorPointerEvent{width: 320, height: 90}, release: make(chan mirrorPointerEvent, 1), ctx: ctx, cancel: cancel}
	owner := &Owner{mirror: &mirrorState{inputBusy: true, gesture: g}}
	up := ui.InputEvent{Kind: ui.InputPointerUp, X: 120, Y: 40}
	if !owner.MirrorPointer(nil, g.row, up, 320, 90) {
		t.Fatal("release escaped to another control")
	}
	select {
	case got := <-g.release:
		if got.event != up {
			t.Fatalf("release changed: %+v", got)
		}
	default:
		t.Fatal("fast release was dropped while down busy")
	}
	owner.MirrorPointer(nil, g.row, up, 320, 90)
	if len(g.release) != 0 {
		t.Fatal("duplicate up queued a second release")
	}
	if ctx.Err() != nil {
		t.Fatal("valid pinned release cancelled")
	}
}
func TestMirrorMovedOrWrongOccurrenceCancelsInsteadOfRetargeting(t *testing.T) {
	for _, wrong := range []struct {
		handle string
		width  float32
		x      float32
	}{{"original-B", 320, 20}, {"original-A", 300, 20}, {"original-A", 320, -1}} {
		ctx, cancel := context.WithCancel(context.Background())
		g := &mirrorGesture{row: MirrorOccurrence{Handle: "original-A"}, down: mirrorPointerEvent{width: 320, height: 90}, release: make(chan mirrorPointerEvent, 1), ctx: ctx, cancel: cancel}
		owner := &Owner{mirror: &mirrorState{inputBusy: true, gesture: g}}
		owner.MirrorPointer(nil, MirrorOccurrence{Handle: wrong.handle}, ui.InputEvent{Kind: ui.InputPointerUp, X: wrong.x, Y: 20}, wrong.width, 90)
		if ctx.Err() == nil || len(g.release) != 0 {
			t.Fatal("changed native seat accepted old release")
		}
		cancel()
	}
}
func TestMirrorGenerationInvalidatesSnapshotAndPinnedGesture(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "owned", Epoch: "1"}
	frame := &MirrorFrame{subject: subject, page: 8, revision: 9, generation: 2}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	owner := &Owner{selected: subject, state: Snapshot{Connected: true, Page: 8, Revision: 9}, mirror: &mirrorState{generation: 2, frame: frame, gesture: &mirrorGesture{ctx: ctx, cancel: cancel}}}
	if owner.MirrorSnapshot() != frame {
		t.Fatal("current frame missing")
	}
	owner.mu.Lock()
	owner.invalidateMirrorLocked()
	owner.mu.Unlock()
	if owner.MirrorSnapshot() != nil || ctx.Err() == nil {
		t.Fatal("generation retained stale pixels or original gesture")
	}
	owner.mirror.frame = frame
	owner.mirror.generation = 2
	owner.state.Revision++
	if owner.MirrorSnapshot() != nil {
		t.Fatal("revision mismatch returned old layout")
	}
}
func TestMirrorGeometryRejectsUnboundedOrAmbiguousOriginalOwners(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "owned", Epoch: "1"}
	g := mirrorGeometry{DocumentKey: "document", Revision: 1, Subject: &subject}
	g.Viewport.W, g.Viewport.H, g.Viewport.Scale = 1000, 900, 1
	g.Owners = []MirrorOccurrence{{Handle: "A", Entry: "entry", Turn: 1, Seq: 2, Rect: MirrorRect{X: 10, Y: 20, W: 400, H: 90}, VisibleRect: MirrorRect{X: 10, Y: 20, W: 400, H: 90}, Connected: true, Visible: true}}
	if !validMirrorGeometry(g, subject) {
		t.Fatal("valid original geometry rejected")
	}
	bad := g
	bad.Owners = append(append([]MirrorOccurrence(nil), g.Owners...), g.Owners[0])
	if validMirrorGeometry(bad, subject) {
		t.Fatal("ambiguous duplicate owner accepted")
	}
	bad = g
	bad.Viewport.W = math.Inf(1)
	if validMirrorGeometry(bad, subject) {
		t.Fatal("unbounded viewport accepted")
	}
	bad = g
	bad.Owners = append([]MirrorOccurrence(nil), g.Owners...)
	bad.Owners[0].Rect.X = 900
	if validMirrorGeometry(bad, subject) {
		t.Fatal("visible crop outside actual viewport accepted")
	}
}
