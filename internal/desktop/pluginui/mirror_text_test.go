package pluginui

import (
	"math"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func textFixture() (*Owner, *MirrorTextClient, *mirrorTextFocus) {
	subject := Subject{Kind: "session", SessionID: "owned", Epoch: "1"}
	focus := &mirrorTextFocus{DocumentKey: "original-document", Subject: subject, Handle: "A", Entry: "entry", Turn: 1, Seq: 2, Focus: "input-original", Value: "A😀中", Selection: mirrorTextSelection{Start: 3, End: 3}, Caret: MirrorRect{X: 40, Y: 50, W: 1, H: 20}}
	state := &mirrorTextState{clients: map[string]*MirrorTextClient{}}
	owner := &Owner{selected: subject, state: Snapshot{Connected: true, Page: 8, Revision: 9}, mirror: &mirrorState{text: state}}
	client := &MirrorTextClient{owner: owner, handle: "A", page: 8, row: MirrorRect{X: 20, Y: 30, W: 200, H: 100}, width: 400, height: 200}
	client.applyFocusLocked(focus)
	state.clients["A"] = client
	return owner, client, focus
}
func TestMirrorTextUTF16QueriesDoNotSplitAstralCharacters(t *testing.T) {
	_, c, _ := textFixture()
	text, r := c.TextForRange(ui.TextInputRange{Start: 2, End: 4})
	if text != "😀中" || r != (ui.TextInputRange{Start: 1, End: 4}) {
		t.Fatalf("UTF16 round/clamp: %q %+v", text, r)
	}
	text, r = c.TextForRange(ui.TextInputRange{Start: -20, End: 1000})
	if text != "A😀中" || r != (ui.TextInputRange{Start: 0, End: 4}) {
		t.Fatalf("UTF16 document clamp: %q %+v", text, r)
	}
	c.optimisticReplaceLocked(ui.TextInputRange{Start: 1, End: 3}, "中文")
	if c.value != "A中文中" || c.selection.Range != (ui.TextInputRange{Start: 3, End: 3}) {
		t.Fatal("marked replacement split original astral character")
	}
}
func TestMirrorTextCaretMapsToNativeRowAndRefusesUnknownRanges(t *testing.T) {
	_, c, _ := textFixture()
	bounds, r, ok := c.BoundsForRange(ui.TextInputRange{Start: 3, End: 3})
	if !ok || bounds != (ui.Rect{X: 40, Y: 40, W: 2, H: 40}) || r.Start != 3 || r.End != 3 {
		t.Fatalf("source/native caret: %+v %+v %t", bounds, r, ok)
	}
	if _, _, ok = c.BoundsForRange(ui.TextInputRange{Start: 0, End: 2}); ok {
		t.Fatal("unknown range claimed actual geometry")
	}
	if _, ok = c.IndexForPoint(ui.Point{X: 160, Y: 100}); ok {
		t.Fatal("distant point claimed a text index")
	}
	if n, ok := c.IndexForPoint(ui.Point{X: 40, Y: 60}); !ok || n != 3 {
		t.Fatal("known caret could not map")
	}
}
func TestMirrorTextAuthorityPinsFocusSessionAndExactTextState(t *testing.T) {
	o, c, f := textFixture()
	job := mirrorTextJob{focus: f, client: c, page: 8, state: o.mirror.text}
	if !o.textCurrent(job) {
		t.Fatal("current focus rejected")
	}
	changed := *f
	changed.Focus = "second-input"
	c.applyFocusLocked(&changed)
	if o.textCurrent(job) {
		t.Fatal("changed original element accepted queued text")
	}
	c.applyFocusLocked(f)
	o.selected.Epoch = "2"
	if o.textCurrent(job) {
		t.Fatal("old Session epoch accepted")
	}
	o.selected = f.Subject
	o.mirror.text = &mirrorTextState{}
	if o.textCurrent(job) {
		t.Fatal("detached text state accepted late acknowledgement")
	}
	o.mirror = nil
	if o.textCurrent(job) {
		t.Fatal("closed mirror accepted queued text")
	}
}
func TestMirrorTextMetadataRejectsInvalidNativeRangesAndGeometry(t *testing.T) {
	_, _, f := textFixture()
	if !f.valid() {
		t.Fatal("valid bounded focus rejected")
	}
	for _, change := range []func(*mirrorTextFocus){func(f *mirrorTextFocus) { f.Selection.Start = -1 }, func(f *mirrorTextFocus) { f.Selection.End = 100 }, func(f *mirrorTextFocus) { f.Selection.Start = 2 }, func(f *mirrorTextFocus) { f.Caret.X = math.NaN() }, func(f *mirrorTextFocus) { f.Marked = &mirrorTextRange{Start: 3, End: 2} }} {
		bad := *f
		change(&bad)
		if bad.valid() {
			t.Fatalf("invalid original text metadata accepted: %+v", bad)
		}
	}
}
func TestMirrorTextWorkerDetachedStateAndKeyMapping(t *testing.T) {
	o, _, _ := textFixture()
	old := o.mirror.text
	old.busy = true
	old.queue = []mirrorTextJob{{}}
	o.mirror = nil
	o.textWorker(old)
	if old.busy {
		t.Fatal("detached worker stayed busy")
	}
	key, code, vk := mirrorKey(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyA, Mods: ui.Shift | ui.Ctrl})
	if key != "A" || code != "KeyA" || vk != 65 {
		t.Fatalf("typed physical key lost: %s %s %d", key, code, vk)
	}
	key, code, vk = mirrorKey(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyBackspace})
	if key != "Backspace" || code != "Backspace" || vk != 8 {
		t.Fatal("editing key became committed text")
	}
	kinds := textTypes(mygo.BrowserTextEvent{Kind: mygo.BrowserTextInsertText})
	if len(kinds) != 4 {
		t.Fatal("text operation lost bounded trusted event kinds")
	}
}
