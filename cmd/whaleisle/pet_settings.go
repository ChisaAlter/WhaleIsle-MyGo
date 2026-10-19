package main

// openPetSettings enters the existing local Pet pane even when Host is offline.
// It keeps drafts and the selected conversation intact.
func (s *mainState) openPetSettings() {
	s.mu.Lock()
	if s.quitting {
		s.mu.Unlock()
		return
	}
	s.settingsEmbeddedSection = ""
	s.settingsOpen = true
	s.launcherOpen = false
	s.usageOpen = false
	s.scheduleOpen = false
	s.imOpen = false
	s.marketOpen = false
	s.remoteOpen = false
	s.webviewOpen = false
	s.browserPreviewOpen = false
	s.assistantConversationGeneration++
	s.assistantOpen = false
	s.toolsMenuOpen = false
	s.settingsDesktopActive = false
	s.settingsAccountActive = false
	s.settingsCredentialsActive = false
	s.settingsAboutActive = false
	s.settingsAppearanceActive = false
	s.settingsInterfaceActive = false
	s.settingsModelsActive = false
	s.settingsMCPActive = false
	s.settingsPluginsActive = false
	s.settingsSkillsActive = false
	s.settingsAgentPresetsActive = false
	settings, win := s.settings, s.win
	s.mu.Unlock()
	if settings != nil {
		settings.Select("pet")
	}
	s.showMainWindow()
	if win != nil {
		win.Update(func() {})
	}
}
