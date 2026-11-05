package main

import (
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
)

// This observer is global: changing Coding Tools must repair future-session
// defaults even when the presets settings page is not open.
func (s *mainState) observeAgentPresetCodingTools() {
	s.mu.Lock()
	source, settings := s.sessions, s.settings
	if source == nil || settings == nil {
		s.mu.Unlock()
		return
	}
	if s.agentPresetManager == nil || !s.agentPresetManager.ForSource(source) {
		s.agentPresetManager = session.NewAgentPresetManager(source)
	}
	manager := s.agentPresetManager
	s.mu.Unlock()
	view := settings.Snapshot()
	ready := view.ClientReady && view.Described && view.HasDocument && !view.Loading && !view.Saving && !view.PreferencePending["ui-settings.enabled"]
	var revision, presetRevision float64
	found, presetFound := false, false
	for _, ns := range view.Namespaces {
		if ns.NS == "ui-settings" {
			revision, found = ns.Revision, true
		}
		if ns.NS == "agent-preset-registry" {
			presetRevision, presetFound = ns.Revision, true
		}
	}
	manager.ObserveCodingTools(ready && found && presetFound, !settingsstore.DeveloperToolsEnabled(view), revision, presetRevision)
}
