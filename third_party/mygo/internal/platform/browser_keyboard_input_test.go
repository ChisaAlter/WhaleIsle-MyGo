package platform

import (
	"strings"
	"testing"
)

func textOffset(v int) *int { return &v }

func TestBrowserTextValidationUnicodeAndRanges(t *testing.T) {
	valid := []BrowserTextEvent{
		{Kind: BrowserTextKeyDown, Key: "A", Code: "KeyA", WindowsVirtualKeyCode: 65, NativeVirtualKeyCode: 65, Modifiers: BrowserTextAlt | BrowserTextCtrl | BrowserTextMeta | BrowserTextShift, Repeat: true},
		{Kind: BrowserTextKeyUp, Key: "Enter", Code: "Enter", WindowsVirtualKeyCode: 13},
		{Kind: BrowserTextInsertText, Text: "中文🙂\n\t"},
		{Kind: BrowserTextInsertText, Text: strings.Repeat("x", 64<<10)},
		{Kind: BrowserTextComposition, Text: "a🙂界", SelectionStart: 1, SelectionEnd: 3},
		{Kind: BrowserTextComposition, Text: "🙂", SelectionStart: 2, SelectionEnd: 2, ReplacementStart: textOffset(0), ReplacementEnd: textOffset(99)},
		{Kind: BrowserTextComposition},
		{Kind: BrowserTextCancelComposition},
	}
	for _, e := range valid {
		if err := ValidateBrowserTextEvent(e); err != nil {
			t.Errorf("valid %+v: %v", e, err)
		}
	}
	invalid := []BrowserTextEvent{
		{Kind: 99},
		{Kind: BrowserTextInsertText, Text: string([]byte{0xff})},
		{Kind: BrowserTextInsertText, Text: strings.Repeat("x", (64<<10)+1)},
		{Kind: BrowserTextKeyDown, Key: "A", Code: "KeyA", Modifiers: 16},
		{Kind: BrowserTextKeyDown, Key: "A", Code: "KeyA", WindowsVirtualKeyCode: 256},
		{Kind: BrowserTextKeyDown, Key: "A", Code: "KeyA", NativeVirtualKeyCode: -1},
		{Kind: BrowserTextKeyDown, Key: "A", Code: "KeyA", NativeVirtualKeyCode: 65536},
		{Kind: BrowserTextKeyDown, Key: "A"},
		{Kind: BrowserTextKeyDown, Key: "\n", Code: "Enter"},
		{Kind: BrowserTextKeyDown, Key: "A", Code: "KeyA", Text: "a"},
		{Kind: BrowserTextKeyUp, Key: "A", Code: "KeyA", Repeat: true},
		{Kind: BrowserTextInsertText, Text: "A", Key: "A"},
		{Kind: BrowserTextInsertText, Text: "A", SelectionEnd: 1},
		{Kind: BrowserTextComposition, Text: "🙂", SelectionStart: 1, SelectionEnd: 2},
		{Kind: BrowserTextComposition, Text: "🙂", SelectionEnd: 1},
		{Kind: BrowserTextComposition, Text: "x", SelectionEnd: 2},
		{Kind: BrowserTextComposition, Text: "x", SelectionStart: 1, SelectionEnd: 0},
		{Kind: BrowserTextComposition, SelectionStart: -1},
		{Kind: BrowserTextComposition, ReplacementStart: textOffset(0)},
		{Kind: BrowserTextComposition, ReplacementEnd: textOffset(0)},
		{Kind: BrowserTextComposition, ReplacementStart: textOffset(-1), ReplacementEnd: textOffset(-1)},
		{Kind: BrowserTextComposition, ReplacementStart: textOffset(2), ReplacementEnd: textOffset(1)},
		{Kind: BrowserTextCancelComposition, Text: "not committed"},
		{Kind: BrowserTextCancelComposition, ReplacementStart: textOffset(0), ReplacementEnd: textOffset(0)},
	}
	for i, e := range invalid {
		if err := ValidateBrowserTextEvent(e); err == nil {
			t.Errorf("invalid %d accepted: %+v", i, e)
		}
	}
}

func TestBrowserTextOptionalProviderAndDetachedOffsets(t *testing.T) {
	if BrowserTextInputProvider(struct{}{}) != nil {
		t.Fatal("missing optional provider advertised support")
	}
	start, end := 3, 8
	original := BrowserTextEvent{Kind: BrowserTextComposition, ReplacementStart: &start, ReplacementEnd: &end}
	clone := CloneBrowserTextEvent(original)
	start, end = 5, 10
	if *clone.ReplacementStart != 3 || *clone.ReplacementEnd != 8 {
		t.Fatal("mutable caller ranges retained")
	}
	if BrowserTextAlt != 1 || BrowserTextCtrl != 2 || BrowserTextMeta != 4 || BrowserTextShift != 8 {
		t.Fatal("modifier mask changed")
	}
}
