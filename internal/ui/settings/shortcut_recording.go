package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
)

func shortcutRecorder(c *ui.Context, e *shortcuts.Editor, id string) {
	if e.Recording != id {
		if ui.Button(c, providerCopy("录制按键…", "Record shortcut…")).Label("record-" + id).Disabled(e.ReadError != "").Clicked() {
			e.BeginRecording(id)
		}
		return
	}
	// A focused custom-input element makes PageShortcut yield. It does not
	// attach a text client, so IME text/composition can never become a binding.
	ui.Text(c, providerCopy("请按组合键，Escape 取消", "Press a chord; Escape cancels")).Key("capture-" + id).Label("capture-" + id).Padding(8).Focusable().AutoFocus().HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputText || ev.Kind == ui.InputCompose || ev.Kind == ui.InputCommand {
			return true
		}
		if ev.Kind != ui.InputKeyDown {
			return false
		}
		if ev.Repeat {
			return true
		}
		if ev.Key == ui.KeyEscape {
			e.CancelRecording()
			return true
		}
		raw, ok := recordedShortcutChord(ev)
		if !ok {
			e.Err = providerCopy("请包含 Mod（Ctrl/Command）和支持的按键", "Use Mod (Ctrl/Command) and a supported key")
			return true
		}
		e.Record(raw)
		return true
	})
	if ui.Button(c, providerCopy("取消录制", "Cancel recording")).Clicked() {
		e.CancelRecording()
	}
}

func recordedShortcutChord(ev ui.InputEvent) (string, bool) {
	if ev.Kind != ui.InputKeyDown || ev.Repeat || ev.Mods&ui.Cmd == 0 || ev.Mods & ^(ui.Cmd|ui.Shift|ui.Alt) != 0 {
		return "", false
	}
	keys := map[ui.Key]string{ui.KeyEnter: "ENTER", ui.KeyComma: ",", ui.KeyBackslash: "\\", ui.KeyBackquote: "`", ui.KeyPeriod: ".", ui.KeySemicolon: ";", ui.KeyQuote: "'", ui.KeyBracketLeft: "[", ui.KeyBracketRight: "]", ui.KeyMinus: "-", ui.KeyEqual: "=", ui.KeySlash: "/"}
	letters := []ui.Key{ui.KeyA, ui.KeyB, ui.KeyC, ui.KeyD, ui.KeyE, ui.KeyF, ui.KeyG, ui.KeyH, ui.KeyI, ui.KeyJ, ui.KeyK, ui.KeyL, ui.KeyM, ui.KeyN, ui.KeyO, ui.KeyP, ui.KeyQ, ui.KeyR, ui.KeyS, ui.KeyT, ui.KeyU, ui.KeyV, ui.KeyW, ui.KeyX, ui.KeyY, ui.KeyZ}
	digits := []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}
	fs := []ui.Key{ui.KeyF1, ui.KeyF2, ui.KeyF3, ui.KeyF4, ui.KeyF5, ui.KeyF6, ui.KeyF7, ui.KeyF8, ui.KeyF9, ui.KeyF10, ui.KeyF11, ui.KeyF12}
	for i, key := range letters {
		keys[key] = string(rune('A' + i))
	}
	for i, key := range digits {
		keys[key] = fmt.Sprint(i)
	}
	for i, key := range fs {
		keys[key] = fmt.Sprintf("F%d", i+1)
	}
	key, ok := keys[ev.Key]
	if !ok {
		return "", false
	}
	return (shortcuts.Chord{Key: key, Shift: ev.Mods&ui.Shift != 0, Alt: ev.Mods&ui.Alt != 0}).String(), true
}
