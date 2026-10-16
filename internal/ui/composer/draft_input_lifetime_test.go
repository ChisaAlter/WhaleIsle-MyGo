package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Mounting must accept the native client, and repainting must not release an
// active IME composition. A fresh adapter pointer each frame passes the first
// condition but fails the second.
func TestNativeDraftInputRetainsCompositionAcrossFrames(t *testing.T) {
	draft := "A"
	client := NewDraftClient(&draft)
	model := Model{Draft: &draft, TextInput: client, UseTextInputClient: true, TypingFx: true, WorkspaceRoot: "fixture"}
	tt := ui.NewTester(func(c *ui.Context) { View(c, model) }, 400, 1000)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	client.SetSelectionUTF16(1, 1)
	input := typingInputClient{client}
	input.SetMarkedText(nil, "候補", ui.TextInputRange{Start: 2, End: 2})
	want := ui.TextInputRange{Start: 1, End: 3}
	for frame := 0; frame < 3; frame++ {
		tt.Frame()
		if got, marked := client.MarkedRange(); !marked || got != want {
			t.Fatalf("frame %d: marked range = %+v, marked = %v; want %+v", frame, got, marked, want)
		}
		if !client.typing.composing || draft != "A候補" {
			t.Fatalf("frame %d: composition interrupted: composing=%v draft=%q", frame, client.typing.composing, draft)
		}
	}

	// Replacing the draft owner must still release the previous composition.
	nextDraft := "next"
	next := NewDraftClient(&nextDraft)
	model.Draft, model.TextInput = &nextDraft, next
	tt.Frame()
	if _, marked := client.MarkedRange(); marked || client.typing.composing {
		t.Fatal("replacing the draft client did not release its old composition")
	}
	if nextDraft != "next" || next.Draft != &nextDraft {
		t.Fatal("replacing the draft client changed the new draft owner or text")
	}
}
