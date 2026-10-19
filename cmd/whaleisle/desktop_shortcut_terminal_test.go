package main

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/frame"
)

func TestPhysicalCommandsPreserveTerminalCustomInputAndIMEOwners(t *testing.T) {
	for _, scope := range []string{"terminal", "custom-input", "native-input"} {
		t.Run(scope, func(t *testing.T) {
			toggles, other := 0, 0
			terminal := shortcuts.Chord{Key: "F9", Modifiers: []string{"Control"}}
			page := shortcuts.Chord{Key: "F10", Modifiers: []string{"Control"}}
			var text string
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					if scope == "native-input" {
						ui.TextInput(c, &text).AutoFocus()
					} else {
						input := ui.Text(c, "actual input owner").Focusable().AutoFocus().HandleInput(func(ui.InputEvent) bool { return true })
						if scope == "terminal" {
							ui.TerminalShortcutScope(input)
						}
					}
					if frame.NativeCommandPressed(c, terminal, "terminal.toggle-fixture", "terminal") {
						toggles++
					}
					if frame.NativeCommandPressed(c, page, "sidebar.toggle-fixture") {
						other++
					}
				})
			}, 420, 220)
			tt.PhysicalKeyDown("F9", ui.KeyF9, ui.Ctrl, false, false, false)
			tt.PhysicalKeyUp("F9", ui.KeyF9, ui.Ctrl, false)
			tt.PhysicalKeyDown("F10", ui.KeyF10, ui.Ctrl, false, false, false)
			tt.PhysicalKeyUp("F10", ui.KeyF10, ui.Ctrl, false)
			want := 0
			if scope == "terminal" {
				want = 1
			}
			if toggles != want || other != 0 {
				t.Fatal("physical command crossed input scope", scope, toggles, other)
			}
			tt.Compose("candidate", 9)
			tt.PhysicalKeyDown("F9", ui.KeyF9, ui.Ctrl, false, true, false)
			tt.PhysicalKeyUp("F9", ui.KeyF9, ui.Ctrl, true)
			if toggles != want {
				t.Fatal("IME/AltGr-guarded input invoked terminal command")
			}
		})
	}
}
