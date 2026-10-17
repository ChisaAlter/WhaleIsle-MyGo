package composer

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"runtime"
	"testing"
)

func TestDraftNavigationGraphemeSelectionAndWord(t *testing.T) {
	d := NewDraftState("one 😀e\u0301 two")
	c := d.Client
	key := func(k ui.Key, mods ui.Modifiers) {
		t.Helper()
		if !draftInputEvent(c, Model{}, ui.InputEvent{Kind: ui.InputKeyDown, Key: k, Mods: mods}) {
			t.Fatalf("unhandled %v", k)
		}
	}
	c.SetSelectionUTF16(4, 4)
	key(ui.KeyRight, 0)
	if c.Selection().Caret() != 6 {
		t.Fatalf("emoji split: %+v", c.Selection())
	}
	key(ui.KeyRight, ui.Shift)
	if c.Selection().Range != (ui.TextInputRange{Start: 6, End: 8}) {
		t.Fatalf("combining selection: %+v", c.Selection())
	}
	key(ui.KeyLeft, 0)
	if c.Selection().Caret() != 6 {
		t.Fatal("left did not collapse selection")
	}
	mods := ui.Ctrl
	if runtime.GOOS == "darwin" {
		mods = ui.Alt
	}
	c.SetSelectionUTF16(ui.UTF16Len(d.Text), ui.UTF16Len(d.Text))
	key(ui.KeyLeft, mods|ui.Shift)
	if c.Selection().Range.Start != 9 || !c.Selection().Reversed {
		t.Fatalf("word selection: %+v", c.Selection())
	}
}

func TestDraftNavigationVerticalAndIME(t *testing.T) {
	d := NewDraftState("abcd\nx\nabcd")
	c := d.Client
	c.SetSelectionUTF16(3, 3)
	for _, want := range []int{6, 10} {
		if !c.navigate(ui.InputEvent{Key: ui.KeyDown}) || c.Selection().Caret() != want {
			t.Fatalf("vertical column want %d: %+v", want, c.Selection())
		}
	}
	c.navigate(ui.InputEvent{Key: ui.KeyHome})
	if c.Selection().Caret() != 7 {
		t.Fatalf("home must be current line: %+v", c.Selection())
	}
	c.navigate(ui.InputEvent{Key: ui.KeyEnd, Mods: ui.Ctrl})
	if c.Selection().Caret() != 11 {
		t.Fatal("ctrl end")
	}
	c.SetMarkedText(nil, "中", ui.TextInputRange{Start: 1, End: 1})
	before := c.Selection()
	if c.navigate(ui.InputEvent{Key: ui.KeyLeft}) || c.Selection() != before {
		t.Fatal("IME navigation stolen")
	}
}

func TestDraftRedoKeyAndCommittedHistoryBoundary(t *testing.T) {
	d := NewDraftState("")
	d.Client.ReplaceText(nil, "sent")
	if !draftInputEvent(d.Client, Model{}, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyZ, Mods: ui.Cmd}) {
		t.Fatal("undo")
	}
	if !draftInputEvent(d.Client, Model{}, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyZ, Mods: ui.Cmd | ui.Shift}) || d.Text != "sent" {
		t.Fatalf("redo %q", d.Text)
	}
	other := NewDraftState("")
	other.Client.ReplaceText(nil, "other")
	f, ok := d.BeginDetachedSend()
	if !ok {
		t.Fatal("send")
	}
	if d.Client.Undo() || d.Client.Redo() || d.Text != "" {
		t.Fatal("sent draft resurrected")
	}
	d.Client.ReplaceText(nil, "new")
	if !d.Client.Undo() || d.Text != "" || !d.Client.Redo() || d.Text != "new" {
		t.Fatal("post-send history broken")
	}
	if !other.Client.Undo() || other.Text != "" {
		t.Fatal("other session history changed")
	}
	d.CompleteDetachedSend(f.ID, nil, errors.New("offline"))
	d.RestoreFailedSends()
	if d.Text != "sent\n\nnew" || d.Client.Undo() || d.Client.Redo() {
		t.Fatalf("restored boundary: %q", d.Text)
	}
}
