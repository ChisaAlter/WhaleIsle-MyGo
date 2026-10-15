package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/shortcuts"
)

// registerPageCommands reuses existing actions. PageShortcut retains text/IME,
// terminal, guest, popup and modal ownership for both defaults and custom keys.
func registerPageCommands(c *ui.Context, m Model) {
	var toggleSidebar func()
	if m.Sizes != nil {
		toggleSidebar = m.Sizes.ToggleSidebar
	}
	create := m.OnCreateSession
	if create == nil && m.Store != nil {
		create = func() { m.Store.CreateSession(context.Background()) }
	}
	actions := map[string]func(){
		"sidebar.toggle": toggleSidebar, "session.new": create, "workspace.open": m.OnBrowseWorkspace,
		"settings.open": m.OnOpenSettings, "tools.toggle": m.OnToggleTools,
		"tools.toggle.alternate": m.OnToggleTools, "terminal.toggle": m.OnToggleTerminal,
	}
	bindings := shortcuts.Resolve(m.PageShortcutBindings)
	for _, command := range shortcuts.Commands() {
		run := actions[command.ID]
		chord, ok := bindings[command.ID]
		if !ok || run == nil {
			continue
		}
		mods, key := nativePageChord(chord)
		if key != ui.KeyUnknown && c.PageShortcut(mods, key) {
			run()
		}
	}
}
func nativePageChord(chord shortcuts.Chord) (ui.Modifiers, ui.Key) {
	mods := ui.Cmd
	if chord.Shift {
		mods |= ui.Shift
	}
	if chord.Alt {
		mods |= ui.Alt
	}
	letters := []ui.Key{ui.KeyA, ui.KeyB, ui.KeyC, ui.KeyD, ui.KeyE, ui.KeyF, ui.KeyG, ui.KeyH, ui.KeyI, ui.KeyJ, ui.KeyK, ui.KeyL, ui.KeyM, ui.KeyN, ui.KeyO, ui.KeyP, ui.KeyQ, ui.KeyR, ui.KeyS, ui.KeyT, ui.KeyU, ui.KeyV, ui.KeyW, ui.KeyX, ui.KeyY, ui.KeyZ}
	digits := []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}
	if len(chord.Key) == 1 {
		if i := strings.Index("ABCDEFGHIJKLMNOPQRSTUVWXYZ", chord.Key); i >= 0 {
			return mods, letters[i]
		}
		if i := strings.Index("0123456789", chord.Key); i >= 0 {
			return mods, digits[i]
		}
	}
	keys := map[string]ui.Key{",": ui.KeyComma, "\\": ui.KeyBackslash, "`": ui.KeyBackquote, ".": ui.KeyPeriod, ";": ui.KeySemicolon, "'": ui.KeyQuote, "[": ui.KeyBracketLeft, "]": ui.KeyBracketRight, "-": ui.KeyMinus, "=": ui.KeyEqual, "/": ui.KeySlash}
	return mods, keys[chord.Key]
}
