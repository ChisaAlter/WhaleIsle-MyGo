package settings

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestShortcutRecorderKeyOwnership(t *testing.T) {
	raw, ok := recordedShortcutChord(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyF5, Mods: ui.Cmd | ui.Shift})
	if !ok || raw != "Mod+Shift+F5" {
		t.Fatal(raw, ok)
	}
	for _, ev := range []ui.InputEvent{{Kind: ui.InputText, Text: "F5"}, {Kind: ui.InputCompose, Text: "拼音"}, {Kind: ui.InputKeyDown, Key: ui.KeyF5}, {Kind: ui.InputKeyDown, Key: ui.KeyF5, Mods: ui.Cmd, Repeat: true}, {Kind: ui.InputKeyDown, Key: ui.KeyEscape, Mods: ui.Cmd}} {
		if _, ok := recordedShortcutChord(ev); ok {
			t.Fatalf("captured %+v", ev)
		}
	}
}
