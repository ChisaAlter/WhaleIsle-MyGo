package pluginui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMirrorNativeLayoutRetainsFocusMetadataButRejectsOldPixelsAndMouse(t *testing.T) {
	o, c, f := textFixture()
	frame := &MirrorFrame{subject: f.Subject, page: 8, revision: 9, generation: 1, Occurrences: []MirrorOccurrence{{Handle: "A", Entry: "entry", Turn: 1, Seq: 2, Visible: true, Rect: MirrorRect{W: 200, H: 100}}}}
	o.state.Outlets = []SessionOutlet{{Handle: "A", EntryID: "entry", Slot: "conversation.chat.turnTail", HasTurn: true, Turn: 1, HasTurnTail: true, TurnTailSeq: 2, Available: true}}
	o.mirror.frame, o.mirror.layout, o.mirror.generation = frame, frame, 1
	o.invalidateMirrorLocked()
	o.state.Revision++
	if o.MirrorLayout() != frame || o.MirrorSnapshot() != nil {
		t.Fatal("capture gap lost native layout or retained old interactive pixels")
	}
	if o.mirrorCurrent(frame) {
		t.Fatal("old mouse frame regained authority")
	}
	client := o.MirrorTextClient(frame, frame.Occurrences[0])
	if client != c || client.focus != f && !client.focus.same(f) {
		t.Fatal("same original text target lost stable native client during capture gap")
	}
	o.state.Outlets = nil
	if o.MirrorLayout() != nil {
		t.Fatal("unmounted original outlet retained native layout")
	}
}
func TestMirrorTextLifecycleRetiresNativeClientsAndQueuedOperations(t *testing.T) {
	o, c, f := textFixture()
	old := o.mirror.text
	old.queue = []mirrorTextJob{{client: c, focus: f, state: old, page: 8}}
	c.pending = 1
	c.marked = &ui.TextInputRange{Start: 1, End: 3}
	o.mu.Lock()
	o.retireMirrorTextLocked()
	o.mu.Unlock()
	if o.mirror.text != nil || !old.closed || len(old.queue) != 0 || c.pending != 0 || c.focus != nil || c.marked != nil {
		t.Fatal("owner boundary retained old native input authority")
	}
	o.textWorker(old)
	if old.busy {
		t.Fatal("retired state could stay attached")
	}
}
func TestMirrorTextForeignReplacementDoesNotChangeCachedOriginalText(t *testing.T) {
	_, c, _ := textFixture()
	before := c.value
	c.ReplaceText(&ui.TextInputRange{Start: 0, End: 1}, "other")
	if c.value != before {
		t.Fatal("refused unrelated replacement changed local text projection")
	}
}
