package main

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/frame"
)

// Application shortcuts are registered on every native route. PageShortcut
// preserves existing editor/terminal/IME/modal ownership; no OS hook is used.
func (s *mainState) applicationCommandActions(c *ui.Context) map[string]func() {
	return map[string]func(){
		"window.zoomIn":     func() { c.SetZoomFactor(c.ZoomFactor() + 0.1) },
		"window.zoomOut":    func() { c.SetZoomFactor(c.ZoomFactor() - 0.1) },
		"window.actualSize": func() { c.SetZoomFactor(1) },
		"window.fullscreen": func() {
			s.mu.Lock()
			win := s.win
			s.mu.Unlock()
			if win != nil {
				win.ToggleFullScreen()
			}
		},
		"shortcuts.open": func() {
			s.selectMainPanel("")
			s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: "interface"})
			c.Invalidate()
		},
	}
}

func (s *mainState) registerApplicationCommands(c *ui.Context) {
	actions := s.applicationMenuActions(c)
	values := s.pageShortcutBindings()
	bindings := shortcuts.Resolve(values)
	accepted, _ := json.Marshal(values)
	for _, command := range shortcuts.Commands() {
		run := actions[command.ID]
		chord, ok := bindings[command.ID]
		if run == nil || !ok {
			continue
		}
		if frame.NativeCommandPressed(c, chord, command.ID+string(accepted)) {
			run()
		}
	}
}
