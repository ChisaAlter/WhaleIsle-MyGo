package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDraftClientInsertDeleteSelection(t *testing.T) {
	draft := "hello"
	c := NewDraftClient(&draft)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: 5, End: 5}}

	c.ReplaceText(nil, "!")
	if draft != "hello!" {
		t.Fatalf("insert: %q", draft)
	}
	if c.selection.Range != (ui.TextInputRange{Start: 6, End: 6}) {
		t.Fatalf("caret after insert: %+v", c.selection)
	}

	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: 1, End: 4}}
	c.ReplaceText(nil, "")
	if draft != "ho!" {
		t.Fatalf("delete selection: %q", draft)
	}

	got, actual := c.TextForRange(ui.TextInputRange{Start: 0, End: 2})
	if got != "ho" || actual != (ui.TextInputRange{Start: 0, End: 2}) {
		t.Fatalf("TextForRange: %q %+v", got, actual)
	}
}

func TestDraftClientUTF16Surrogates(t *testing.T) {
	draft := "A😀cafe"
	c := NewDraftClient(&draft)
	end := ui.UTF16Len(draft)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}

	c.ReplaceText(&ui.TextInputRange{Start: 6, End: 7}, "é")
	if draft != "A😀café" {
		t.Fatalf("utf16 replace: %q", draft)
	}

	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: 6, End: 7}}
	c.SetMarkedText(nil, "e\u0301", ui.TextInputRange{Start: 2, End: 2})
	if draft != "A😀cafe\u0301" || c.marked == nil {
		t.Fatalf("preedit text: %q mark=%v", draft, c.marked)
	}
	if c.selection.Range != (ui.TextInputRange{Start: 8, End: 8}) {
		t.Fatalf("preedit caret utf16: %+v", c.selection)
	}

	c.ReplaceText(nil, "é")
	if draft != "A😀café" || c.marked != nil {
		t.Fatalf("commit: %q", draft)
	}
}

func TestDraftClientSyncExternalEdit(t *testing.T) {
	draft := "abc"
	c := NewDraftClient(&draft)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: 3, End: 3}}

	draft = "chip removed"
	c.Sync()
	if c.selection.Range.Start > ui.UTF16Len(draft) || c.selection.Range.End > ui.UTF16Len(draft) {
		t.Fatalf("selection not clamped: %+v len=%d", c.selection, ui.UTF16Len(draft))
	}
	if c.marked != nil {
		t.Fatal("marked should clear on external edit")
	}
}

func TestDraftClientUndoRedo(t *testing.T) {
	draft := "ab"
	c := NewDraftClient(&draft)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: 2, End: 2}}

	c.ReplaceText(nil, "c")
	if draft != "abc" {
		t.Fatalf("after insert: %q", draft)
	}
	if !c.Undo() {
		t.Fatal("undo failed")
	}
	if draft != "ab" || c.selection.Range != (ui.TextInputRange{Start: 2, End: 2}) {
		t.Fatalf("after undo: %q sel=%+v", draft, c.selection)
	}
	if !c.Redo() {
		t.Fatal("redo failed")
	}
	if draft != "abc" {
		t.Fatalf("after redo: %q", draft)
	}
}

func TestDraftClientUndoMarkedText(t *testing.T) {
	draft := "hi"
	c := NewDraftClient(&draft)
	end := ui.UTF16Len(draft)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}

	c.SetMarkedText(nil, "x", ui.TextInputRange{Start: 1, End: 1})
	if draft != "hix" || c.marked == nil {
		t.Fatalf("preedit: %q mark=%v", draft, c.marked)
	}
	if !c.Undo() {
		t.Fatal("undo preedit failed")
	}
	if draft != "hi" || c.marked != nil {
		t.Fatalf("after undo preedit: %q mark=%v", draft, c.marked)
	}
}

func TestDraftClientSetSelectionUTF16(t *testing.T) {
	draft := "012345"
	c := NewDraftClient(&draft)
	c.SetSelectionUTF16(2, 4)
	if c.selection.Range != (ui.TextInputRange{Start: 2, End: 4}) {
		t.Fatalf("selection: %+v", c.selection)
	}
}
