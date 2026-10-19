package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
)

func TestNativeIMECommitIsOneUndoAcrossUnmarkOrdering(t *testing.T) {
	for _, unmarkFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit-marked", true: "unmark-explicit-commit"}[unmarkFirst], func(t *testing.T) {
			draft := "A😀旧字Z"
			client := NewDraftClient(&draft)
			input := typingInputClient{client}
			original := ui.TextInputRange{Start: 3, End: 5}
			client.SetSelectionUTF16(original.Start, original.End)
			input.SetMarkedText(nil, "zhong", ui.TextInputRange{Start: 5, End: 5})
			input.SetMarkedText(nil, "中文候选", ui.TextInputRange{Start: 4, End: 4})
			marked, _ := client.MarkedRange()
			if unmarkFirst {
				input.UnmarkText()
				input.ReplaceText(&marked, "中文")
			} else {
				input.ReplaceText(nil, "中文")
			}
			if draft != "A😀中文Z" {
				t.Fatalf("commit: %q", draft)
			}
			if !client.Undo() || draft != "A😀旧字Z" || client.Selection().Range != original {
				t.Fatalf("one undo did not restore document/selection: %q %+v", draft, client.Selection())
			}
			if _, marked := client.MarkedRange(); marked {
				t.Fatal("undo revived a candidate composition")
			}
			if client.Undo() {
				t.Fatal("intermediate candidates leaked undo steps")
			}
			if !client.Redo() || draft != "A😀中文Z" {
				t.Fatal("redo failed to restore committed text")
			}
		})
	}
}

func TestNativeIMEClosingEnterGuardExpiresAndPasteHasOwnUndo(t *testing.T) {
	draft := ""
	client := NewDraftClient(&draft)
	submitted := 0
	m := Model{Draft: &draft, OnSubmitGesture: func(bool) { submitted++ }}
	client.SetMarkedText(nil, "中文", ui.TextInputRange{Start: 2, End: 2})
	client.UnmarkText()
	// Set a controlled instant instead of depending on a ten-millisecond
	// scheduler race or sleeping to test the closing native key interval.
	client.compositionEnded = time.Now().Add(time.Hour)
	enter := ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter}
	if draftInputEvent(client, m, enter) || submitted != 0 {
		t.Fatal("IME closing Enter submitted")
	}
	client.compositionEnded = time.Now().Add(-20 * time.Millisecond)
	if !draftInputEvent(client, m, enter) || submitted != 1 {
		t.Fatal("expired IME guard swallowed next Enter")
	}
	client.ClearHistory()
	client.ReplaceText(nil, "a")
	client.ReplaceText(nil, "b")
	at := client.Selection().Range
	client.ReplaceText(&at, "paste\n中文")
	if !client.Undo() || draft != "中文ab" {
		t.Fatalf("paste merged into typing: %q", draft)
	}
	if !client.Undo() || draft != "中文" {
		t.Fatalf("adjacent typing was not one undo: %q", draft)
	}
}
