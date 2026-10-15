package main

import (
	"errors"
	"whaleisle/internal/shortcuts"
)

// The editor and dispatch share one committed snapshot. No Host, Electron
// keybindings file or live keyboard is contacted to edit native page bindings.
func (s *mainState) pageShortcutEditor() *shortcuts.Editor {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shortcutEditor != nil {
		return s.shortcutEditor
	}
	cfg := s.cfgStore
	if cfg == nil {
		return shortcuts.NewEditor(nil, errors.New("configuration unavailable"), nil)
	}
	saved, err := cfg.PageShortcuts(shortcuts.IDs())
	s.shortcutEditor = shortcuts.NewEditor(saved, err, cfg.SavePageShortcuts)
	return s.shortcutEditor
}
func (s *mainState) pageShortcutBindings() map[string]string { return s.pageShortcutEditor().Active() }
