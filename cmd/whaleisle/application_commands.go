package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/frame"
)

// Application shortcuts are registered on every native route. Named command
// scopes arbitrate native input and modal ownership; no OS hook is used.
func (s *mainState) applicationCommandActions(c *ui.Context) map[string]func() {
	return map[string]func(){
		"window.zoomIn":     func() { c.SetZoomFactor(c.ZoomFactor() + 0.1) },
		"window.zoomOut":    func() { c.SetZoomFactor(c.ZoomFactor() - 0.1) },
		"window.actualSize": func() { c.SetZoomFactor(1) },
		"window.fullscreen": func() {
			s.mu.Lock()
			win := mygo.FocusedWindow()
			s.mu.Unlock()
			if win != nil {
				win.ToggleFullScreen()
			}
		},
		"shortcuts.open": func() {
			if c.ShortcutModalName() == "shortcuts" {
				c.CloseTopDialog()
				return
			}
			// Already-open settings retain their current section and drafts.
			editor := s.pageShortcutEditor()
			if c.ShortcutModalName() != "settings" {
				s.selectMainPanel("")
				s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings})
			}
			editor.ReferenceOpen = true
			c.Invalidate()
		},
	}
}

func (s *mainState) registerApplicationCommands(c *ui.Context) {
	// Fixed input excludes terminal/IME/repeat at the native input owner. It
	// remains available while typing or editing font settings, as in 0.3.13.
	s.mu.Lock()
	editor := s.shortcutEditor
	quitting := s.quitting
	s.mu.Unlock()
	// Recording and reset decisions own input before any application command,
	// including the binding which normally opens/closes shortcut help.
	if quitting || editor != nil && (editor.Recording != "" || editor.ResetPending) {
		return
	}
	for _, fixed := range []struct {
		code  string
		mods  ui.Modifiers
		delta int
		id    string
	}{
		{"Equal", ui.Cmd, 1, "fontSize.increase"},
		{"Equal", ui.Cmd | ui.Shift, 1, "fontSize.increase.shift"},
		{"Minus", ui.Cmd, -1, "fontSize.decrease"},
	} {
		if c.PhysicalShortcut(fixed.mods, fixed.code, "", fixed.id, "application-fixed") {
			s.adjustFontSizes(fixed.delta)
		}
	}
	actions := s.applicationMenuActions(c)
	bindings := s.pageShortcutChords()
	accepted := s.pageShortcutEncoded()
	for _, command := range shortcuts.Commands() {
		run := actions[command.ID]
		chord, ok := bindings[command.ID]
		if run == nil || !ok {
			continue
		}
		if frame.NativeCommandPressed(c, chord, command.ID+accepted, shortcuts.NativeCommandScope(command.ID)) {
			run()
		}
	}
}
