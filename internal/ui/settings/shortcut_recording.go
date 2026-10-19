package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"runtime"
	"sort"
	"whaleisle/internal/shortcuts"
)

func shortcutRecorder(c *ui.Context, e *shortcuts.Editor, id string) {
	desktopChords := runtime.GOOS == "windows" || runtime.GOOS == "darwin"
	if e.Recording != id {
		if ui.Button(c, providerCopy("录制按键…", "Record shortcut…")).Label("record-" + id).Disabled(e.ReadError != "" || e.ResetPending).Clicked() {
			e.BeginRecording(id)
		}
		return
	}
	anchor := ui.Text(c, providerCopy("按住一至两个键，释放后保存", "Hold one or two keys; release to save")).Key("capture-" + id).Label("capture-" + id).Padding(8).Focusable().AutoFocus()
	state := ui.Local(anchor, "physical-recording", func() physicalRecording { return physicalRecording{} })
	reset := func() { state.held = nil; state.pending = nil; state.blocked = false }
	capture := func() {
		if state.pending == nil {
			return
		}
		binding := *state.pending
		state.pending = nil
		state.blocked = true
		e.RecordAndSave(binding.String())
	}
	anchor.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputBlur || ev.Kind == ui.InputCompose || ev.ShortcutBlocked {
			reset()
			return true
		}
		if ev.Kind == ui.InputText || ev.Kind == ui.InputCommand {
			return true
		}
		if ev.Kind == ui.InputModifiers {
			if state.pending != nil && ev.Mods&state.mods != state.mods {
				capture()
			} else {
				reset()
			}
			state.mods = ev.Mods
			return true
		}
		if ev.Kind == ui.InputKeyUp {
			delete(state.held, ev.Code)
			if state.pending != nil && (shortcuts.PhysicalCode(state.pending.Key) == ev.Code || shortcuts.PhysicalCode(state.pending.SecondKey) == ev.Code) {
				capture()
			}
			if len(state.held) == 0 {
				state.blocked = false
			}
			return true
		}
		if ev.Kind != ui.InputKeyDown {
			return false
		}
		if ev.Repeat {
			return true
		}
		if ev.Code == "" {
			reset()
			e.Err = providerCopy("此按键没有可用的物理键码", "Physical key code unavailable")
			return true
		}
		if state.held == nil {
			state.held = map[string]bool{}
		}
		state.held[ev.Code] = true
		if state.blocked {
			return true
		}
		if len(state.held) > 2 {
			state.pending = nil
			state.blocked = true
			e.Err = providerCopy("最多同时按住两个键", "Hold at most two keys")
			return true
		}
		mods := []string{}
		for _, m := range []struct {
			bit  ui.Modifiers
			name string
		}{{ui.Ctrl, "Control"}, {ui.Alt, "Alt"}, {ui.Shift, "Shift"}, {ui.Super, "Meta"}} {
			if ev.Mods&m.bit != 0 {
				mods = append(mods, m.name)
			}
		}
		keys := []string{}
		for code := range state.held {
			keys = append(keys, code)
		}
		sort.Strings(keys)
		if !desktopChords {
			keys = []string{ev.Code}
		}
		chord := shortcuts.Chord{Key: shortcuts.PhysicalKey(keys[0]), Modifiers: mods}
		if len(keys) == 2 {
			chord.SecondKey = shortcuts.PhysicalKey(keys[1])
		}
		state.pending = &chord
		state.mods = ev.Mods
		e.Err = ""
		return true
	})
	if !anchor.Focused() {
		reset()
	}
	if e.Err != "" {
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, providerCopy("审阅当前配置，保留录制", "Review current settings; keep recording")).Clicked() {
				reset()
				e.Review()
			}
			if ui.Button(c, providerCopy("重试保存当前录制", "Retry saving the retained recording")).Clicked() {
				e.SaveRow(id, false)
			}
		})
	}
	if ui.Button(c, providerCopy("取消录制", "Cancel recording")).Clicked() {
		reset()
		e.CancelRecording()
	}
}

type physicalRecording struct {
	held    map[string]bool
	pending *shortcuts.Chord
	mods    ui.Modifiers
	blocked bool
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
