package main

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/shortcuts"
)

// The page owns its own command registry. This fallback contains only native
// application navigation, never arbitrary Host calls or session mutations.
func webApplicationFallbacks(bindings map[string]shortcuts.Chord, dispatch func(string)) []mygo.WebApplicationShortcut {
	ids := []string{"settings.open", "shortcuts.open", "market.open", "launcher.open", "settings.mcp", "settings.skills", "settings.about"}
	var out []mygo.WebApplicationShortcut
	for _, id := range ids {
		chord, ok := bindings[id]
		b := chord.PhysicalBinding()
		if !ok || b == nil || chord.SecondKey != "" || len(b.Modifiers) == 0 || browserOwnsChord(chord) {
			continue
		}
		primary := false
		unsupported := false
		for _, mod := range b.Modifiers {
			switch mod {
			case "primary", "control":
				primary = true
			case "shift", "alt":
			default:
				unsupported = true
			}
		}
		if !primary || unsupported {
			continue
		}
		command := id
		out = append(out, mygo.WebApplicationShortcut{Key: chord.Key, Ctrl: true, Alt: chord.Alt, Shift: chord.Shift, Run: func() { dispatch(command) }})
	}
	return out
}
func browserOwnsChord(c shortcuts.Chord) bool {
	// Keep Chromium's own navigation, search, print, tabs, downloads and zoom.
	// Since the observer never cancels browser defaults, these cannot also run
	// an application action. Rebinding to a free chord enables the fallback.
	if c.Alt {
		return c.Key == "LEFT" || c.Key == "RIGHT" || c.Key == "HOME"
	}
	if c.Shift {
		switch c.Key {
		case "N", "T", "W", "P", "O", "B", "J", "I", "C", "DELETE", "TAB":
			return true
		}
		return false
	}
	switch c.Key {
	case "N", "T", "W", "P", "F", "G", "L", "O", "J", "H", "R", "D", "B", "U", "S", "=", "-", "0", "TAB", "F5":
		return true
	}
	return false
}
func (s *mainState) syncWebApplicationShortcuts(bindings map[string]shortcuts.Chord) {
	revision := s.pageShortcutPreferences().Accepted().Revision
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win == nil {
		return
	}
	win.SetWebApplicationShortcuts(webApplicationFallbacks(bindings, func(id string) {
		if s.applicationMenu == nil {
			return
		}
		s.applicationMenu.pending = append(s.applicationMenu.pending, desktopMenuGesture{id, revision})
		win.Update(func() {})
	}))
}
