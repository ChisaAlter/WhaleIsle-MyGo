package shortcuts

import (
	"fmt"
	"strings"
	"whaleisle/internal/shortcutprefs"
)

// Same-action IDs confirmed against e9 ui-workspace/shortcuts.ts,
// ui-layout/index.ts, ui-titlebar/apply.ts and ui-shortcuts/index.ts.
var sharedIDs = map[string]string{
	"sidebar.toggle": "sidebar.left.toggle", "tools.toggle": "surfaces.toggle", "terminal.toggle": "terminal.drawer.toggle",
	"settings.open": "settings.open", "session.new": "session.new", "session.search": "session.search", "session.rename": "session.rename", "session.fork": "session.fork", "session.archive": "session.archive", "workspace.add": "workspace.add", "shortcuts.open": "shortcuts.open",
}

func CanonicalID(id string) string {
	if c := sharedIDs[id]; c != "" {
		return c
	}
	return "mygo." + id
}
func SharedDefaults() map[string]string {
	d := Defaults()
	d["terminal.toggle"] = "Mod+`"
	d["session.search"] = "Mod+K"
	d["session.rename"] = "Mod+Alt+G"
	d["session.fork"] = "Mod+Alt+F"
	d["session.archive"] = "Mod+Shift+A"
	d["workspace.add"] = "Mod+O"
	// Compatibility rows remain editable, but do not introduce a second default owner.
	d["workspace.open"] = ""
	d["tools.toggle.alternate"] = ""
	return d
}

var physicalCodes = map[string]string{",": "Comma", ".": "Period", "/": "Slash", "\\": "Backslash", "`": "Backquote", ";": "Semicolon", "'": "Quote", "[": "BracketLeft", "]": "BracketRight", "-": "Minus", "=": "Equal", "ENTER": "Enter"}

func SharedBinding(raw string) (*shortcutprefs.Binding, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	// Legacy saved native values are explicit overrides, even equal to old defaults.
	parts := strings.Split(raw, "+")
	b := &shortcutprefs.Binding{Modifiers: []string{}}
	for _, p := range parts[:len(parts)-1] {
		switch strings.ToUpper(p) {
		case "MOD", "CTRL", "CONTROL", "CMD", "COMMAND":
			b.Modifiers = append(b.Modifiers, "primary")
		case "SHIFT":
			b.Modifiers = append(b.Modifiers, "shift")
		case "ALT":
			b.Modifiers = append(b.Modifiers, "alt")
		default:
			return nil, fmt.Errorf("unsupported modifier %s", p)
		}
	}
	key := strings.ToUpper(parts[len(parts)-1])
	b.Code = physicalCodes[key]
	if b.Code == "" {
		if len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
			b.Code = "Key" + key
		} else if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			b.Code = "Digit" + key
		} else if functionKey(key) {
			b.Code = key
		} else {
			return nil, fmt.Errorf("unsupported key %s", key)
		}
	}
	return b, nil
}
func SharedDefinitions(profile string) []shortcutprefs.Definition {
	var defs []shortcutprefs.Definition
	for id, v := range SharedDefaults() {
		d := shortcutprefs.Definition{ID: CanonicalID(id), Defaults: map[string]shortcutprefs.Binding{}}
		if b, _ := SharedBinding(v); b != nil {
			d.Defaults[profile] = *b
		}
		defs = append(defs, d)
	}
	return defs
}
func SharedDisplay(b *shortcutprefs.Binding, platform string) string {
	if b == nil {
		return ""
	}
	key := strings.TrimPrefix(strings.TrimPrefix(b.Code, "Key"), "Digit")
	for k, c := range physicalCodes {
		if c == b.Code {
			key = k
		}
	}
	var mods []string
	primary := "control"
	if platform == "macos" {
		primary = "meta"
	}
	for _, m := range b.Modifiers {
		switch m {
		case "primary":
			mods = append(mods, "Mod")
		case primary:
			mods = append(mods, "Mod")
		case "shift":
			mods = append(mods, "Shift")
		case "alt":
			mods = append(mods, "Alt")
		default:
			mods = append(mods, m)
		}
	}
	if b.SecondCode != "" {
		key += " & " + b.SecondCode
	}
	return strings.Join(append(mods, key), "+")
}
