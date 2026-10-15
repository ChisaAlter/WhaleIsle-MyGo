package main

import "whaleisle/internal/desktop/mcpconfig"

// Run at the end of each native frame so a tab/close click in that same frame
// stops polling. rendered also excludes other pages that return before settings.
func (s *mainState) observeMCPHealth(rendered bool) {
	s.mu.Lock()
	visible := rendered && s.settingsOpen && s.settingsMCPActive && !s.quitting
	win := s.win
	s.mu.Unlock()
	if win == nil || !win.IsVisible() || win.IsMinimized() {
		visible = false
	}
	s.dshMu.Lock()
	defer s.dshMu.Unlock()
	if controller := s.settingsMCPController; controller != nil {
		var caller mcpconfig.Caller
		if visible && s.dshClient != nil {
			caller = s.dshClient
		}
		controller.ObserveHealthRefresh(visible, caller, s.updateMCPSettingsUI)
	}
}

func (s *mainState) suspendMCPHealth() {
	s.dshMu.Lock()
	defer s.dshMu.Unlock()
	if controller := s.settingsMCPController; controller != nil {
		controller.StopHealthRefresh()
	}
}

// Called with dshMu held at both Host attach and detach. Cancel the old list
// before publishing the replacement; the old loop keeps its captured caller.
func (s *mainState) replaceMCPControllerLocked() {
	if controller := s.settingsMCPController; controller != nil {
		controller.StopHealthRefresh()
	}
	s.settingsMCPController = &mcpconfig.Controller{}
}
