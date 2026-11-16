package composer

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
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
	// The first Enter after UnmarkText closes the IME stroke regardless of frame
	// latency; once consumed, the subsequent Enter submits normally.
	enter := ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Code: "Enter"}
	if draftInputEvent(client, m, enter) || submitted != 0 {
		t.Fatal("IME closing Enter submitted")
	}
	if !draftInputEvent(client, m, enter) || submitted != 1 {
		t.Fatal("subsequent Enter after consumed IME guard did not submit")
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

func TestNativeIMEProcessKeyLatchesAndSuggestionGuardAndExternalCancel(t *testing.T) {
	// 1. VK_PROCESSKEY (KeyUnknown + ShortcutBlocked) on Enter latches across
	// UnmarkText/ReplaceText regardless of delay until closing Enter/KeyUp releases.
	draft := ""
	client := NewDraftClient(&draft)
	submitted := 0
	m := Model{
		Draft:           &draft,
		TextInput:       client,
		OnSubmitGesture: func(bool) { submitted++ },
		FilePaths:       []string{"readme.md"},
		Snapshot:        session.ViewState{ClientReady: true, SelectedID: "s1"},
	}
	client.SetMarkedText(nil, "@r", ui.TextInputRange{Start: 2, End: 2})
	if _, _, _, _, open := activeSuggestion(m, client); open {
		t.Fatal("activeSuggestion opened during marked preedit")
	}
	if draftInputEvent(client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyUnknown, Code: "Enter", ShortcutBlocked: true}) {
		t.Fatal("VK_PROCESSKEY was claimed")
	}
	client.UnmarkText()
	client.ReplaceText(&ui.TextInputRange{Start: 0, End: 2}, "readme")
	if !client.composing() {
		t.Fatal("expected closing Enter process-key latch to remain armed before KeyUp/Enter")
	}
	if _, _, _, _, open := activeSuggestion(m, client); open {
		t.Fatal("activeSuggestion opened during closing IME latch")
	}
	// Closing KeyDown(Enter) is suppressed; KeyUp clears latch; next Enter submits.
	if draftInputEvent(client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Code: "Enter"}) || submitted != 0 {
		t.Fatal("delayed closing Enter after VK_PROCESSKEY submitted")
	}
	draftInputEvent(client, m, ui.InputEvent{Kind: ui.InputKeyUp, Key: ui.KeyEnter, Code: "Enter"})
	if !draftInputEvent(client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Code: "Enter"}) || submitted != 1 {
		t.Fatal("fresh Enter after KeyUp failed to submit")
	}

	// 2. Committing via Space (VK_PROCESSKEY Space -> ReplaceText) must NOT block
	// an immediate subsequent Enter to send.
	client.SetMarkedText(nil, "ni", ui.TextInputRange{Start: 2, End: 2})
	draftInputEvent(client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyUnknown, Code: "Space", ShortcutBlocked: true})
	client.ReplaceText(nil, "你")
	if client.composing() {
		t.Fatal("Space candidate commit kept Enter guard active")
	}
	if !draftInputEvent(client, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter, Code: "Enter"}) || submitted != 2 {
		t.Fatal("Enter after Space commit did not submit immediately")
	}

	// 3. Programmatic *Draft change inside ensure() drops active composition AND
	// swaps textInputHandle() identity so host SetTextInput cancels OS IMM32.
	extDraft := "hi"
	extClient := NewDraftClient(&extDraft)
	extModel := Model{Draft: &extDraft, TextInput: extClient, UseTextInputClient: true, WorkspaceRoot: "fixture"}
	tt := ui.NewTester(func(c *ui.Context) { View(c, extModel) }, 400, 300)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	firstHandle := extClient.textInputHandle()
	extClient.textInputHandle().SetMarkedText(nil, "zhong", ui.TextInputRange{Start: 5, End: 5})
	tt.Frame()
	if extClient.textInputHandle() != firstHandle {
		t.Fatal("normal repaint swapped TextInputClient identity")
	}
	extDraft = "reset"
	tt.Frame()
	if _, marked := extClient.MarkedRange(); marked {
		t.Fatal("ensure() left marked range active after external text mutation")
	}
	if extClient.textInputHandle() == firstHandle {
		t.Fatal("ensure() did not rotate TextInputClient handle to cancel OS IME")
	}
}
