package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"maps"
	"strings"
	"sync"
	"whaleisle/internal/shortcuts"
)

type resolvedPageCommand struct {
	id         string
	identity   string
	region     string
	mods       ui.Modifiers
	code       string
	secondCode string
}

var pageCommandsMemo struct {
	mu      sync.Mutex
	valid   bool
	input   map[string]string
	entries []resolvedPageCommand
}

func resolvedPageCommands(overrides map[string]string) []resolvedPageCommand {
	pageCommandsMemo.mu.Lock()
	if pageCommandsMemo.valid && maps.Equal(pageCommandsMemo.input, overrides) {
		entries := pageCommandsMemo.entries
		pageCommandsMemo.mu.Unlock()
		return entries
	}
	pageCommandsMemo.mu.Unlock()

	bindings := shortcuts.Resolve(overrides)
	accepted, _ := json.Marshal(overrides)
	acceptedStr := string(accepted)
	cmds := shortcuts.Commands()
	entries := make([]resolvedPageCommand, 0, len(cmds))
	for _, command := range cmds {
		chord, ok := bindings[command.ID]
		if !ok {
			continue
		}
		mods, code, secondCode, valid := physicalBindingSpec(chord)
		if !valid {
			continue
		}
		region := ""
		if command.ID == "terminal.toggle" {
			region = "terminal"
		}
		entries = append(entries, resolvedPageCommand{
			id:         command.ID,
			identity:   command.ID + acceptedStr,
			region:     region,
			mods:       mods,
			code:       code,
			secondCode: secondCode,
		})
	}

	pageCommandsMemo.mu.Lock()
	pageCommandsMemo.valid = true
	pageCommandsMemo.input = maps.Clone(overrides)
	pageCommandsMemo.entries = entries
	pageCommandsMemo.mu.Unlock()
	return entries
}

func physicalBindingSpec(chord shortcuts.Chord) (ui.Modifiers, string, string, bool) {
	b := chord.PhysicalBinding()
	if b == nil {
		return 0, "", "", false
	}
	mods := ui.Modifiers(0)
	for _, m := range b.Modifiers {
		switch m {
		case "primary":
			mods |= ui.Cmd
		case "control":
			mods |= ui.Ctrl
		case "alt":
			mods |= ui.Alt
		case "shift":
			mods |= ui.Shift
		case "meta":
			mods |= ui.Super
		}
	}
	return mods, b.Code, b.SecondCode, true
}

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
	addBusinessCommands(c, m, actions)
	guardMainChildCommands(m, actions)
	for _, entry := range resolvedPageCommands(m.PageShortcutBindings) {
		run := actions[entry.id]
		if run == nil {
			continue
		}
		if c.PhysicalShortcut(entry.mods, entry.code, entry.secondCode, entry.identity, entry.region) {
			run()
		}
	}
}

func NativeCommandPressed(c *ui.Context, chord shortcuts.Chord, identity ...string) bool {
	mods, code, secondCode, ok := physicalBindingSpec(chord)
	if !ok {
		return false
	}
	return c.PhysicalShortcut(mods, code, secondCode, identity...)
}

func NativeCommandChord(chord shortcuts.Chord) (ui.Modifiers, ui.Key) {
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
	functionKeys := []ui.Key{ui.KeyF1, ui.KeyF2, ui.KeyF3, ui.KeyF4, ui.KeyF5, ui.KeyF6, ui.KeyF7, ui.KeyF8, ui.KeyF9, ui.KeyF10, ui.KeyF11, ui.KeyF12}
	for index, key := range functionKeys {
		if chord.Key == fmt.Sprintf("F%d", index+1) {
			return mods, key
		}
	}
	keys := map[string]ui.Key{"ENTER": ui.KeyEnter, ",": ui.KeyComma, "\\": ui.KeyBackslash, "`": ui.KeyBackquote, ".": ui.KeyPeriod, ";": ui.KeySemicolon, "'": ui.KeyQuote, "[": ui.KeyBracketLeft, "]": ui.KeyBracketRight, "-": ui.KeyMinus, "=": ui.KeyEqual, "/": ui.KeySlash}
	return mods, keys[chord.Key]
}
