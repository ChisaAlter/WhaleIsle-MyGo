package remote

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestRemoteTextBufferCapturesCommittedNativeInputWithoutFrame(t *testing.T) {
	source := newFileTextBuffer("abcd").(*fileTextBuffer)
	frames := 0
	tt := ui.NewTester(func(c *ui.Context) { frames++; ui.TextAreaBuffer(c, source.buffer).Fill().AutoFocus() }, 320, 180)
	client := tt.NativeTextInputClient()
	if client == nil {
		t.Fatal("native input client missing")
	}
	before := frames
	end := ui.TextInputRange{Start: 4, End: 4}
	client.ReplaceText(&end, "!")
	if got := source.CommittedText(); got != "abcd!" || frames != before {
		t.Fatal("capture waited for another widget frame", got, frames, before)
	}
	selection := ui.TextInputRange{Start: 1, End: 3}
	client.SetMarkedText(&selection, "candidate", ui.TextInputRange{Start: 9, End: 9})
	if got := source.CommittedText(); got != "abcd!" {
		t.Fatal("marked IME replacement entered save capture", got)
	}
	tt.Frame()
	if got := source.CommittedText(); got != "abcd!" {
		t.Fatal("widget frame lost preedit origin", got)
	}
	client.ReplaceText(nil, "committed")
	if got := source.CommittedText(); got != "acommittedd!" {
		t.Fatal("real native commit not captured", got)
	}
}

func TestRemoteTextBufferExplicitReplacementRetiresOldPreedit(t *testing.T) {
	source := newFileTextBuffer("abcd").(*fileTextBuffer)
	tt := ui.NewTester(func(c *ui.Context) { ui.TextAreaBuffer(c, source.buffer).Fill().AutoFocus() }, 320, 180)
	client := tt.NativeTextInputClient()
	if client == nil {
		t.Fatal("native input client missing")
	}
	selection := ui.TextInputRange{Start: 1, End: 3}
	client.SetMarkedText(&selection, "candidate", ui.TextInputRange{Start: 9, End: 9})
	source.ReplaceText("discarded to server content")
	client.UnmarkText()
	if got := source.CommittedText(); got != "discarded to server content" {
		t.Fatal("old IME callback undid explicit discard", got)
	}
}
