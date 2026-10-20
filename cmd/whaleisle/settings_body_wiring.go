package main

import (
	"context"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/config"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/settings"
)

func (s *mainState) settingsPetActiveLocked() bool {
	if !s.settingsOpen || s.settingsEmbeddedSection != "" || s.settings == nil ||
		s.settingsDesktopActive || s.settingsAccountActive || s.settingsCredentialsActive ||
		s.settingsAboutActive || s.settingsAppearanceActive || s.settingsInterfaceActive ||
		s.settingsModelsActive || s.settingsMCPActive || s.settingsPluginsActive ||
		s.settingsSkillsActive || s.settingsAgentPresetsActive {
		return false
	}
	ns := s.settings.Snapshot().SelectedNS
	return ns == "pet" || ns == "live2d"
}

func (s *mainState) wireSettingsBodies(m *settings.Model) {
	if m.Store == nil {
		return
	}
	snapshot, owner, write := m.Store.BindDescribedPreferences()
	access := &settings.BodySettingsAccess{Snapshot: snapshot, Owner: owner, OnChange: func(ns, key, value string) {
		write(context.Background(), ns, map[string]string{key: value})
	}, OnPatch: func(ns string, values map[string]string) {
		write(context.Background(), ns, values)
	}}
	if m.Desktop != nil {
		m.Desktop.Host = access
		m.Desktop.Shortcuts = s.pageShortcutEditor()
		m.Desktop.Preferences = s.themeFontEditor(m.DesktopActive || m.AppearanceActive)
		m.Desktop.Version = s.harnessVersionSnapshot()
		s.mu.Lock()
		m.Desktop.UpdateBusy = s.updateCheckPending || s.nativeReleases.busy
		s.mu.Unlock()
		m.Desktop.AutoStartDesktop = &s.desktopDraft.autoStartDesktop
	}
	if m.Appearance != nil {
		m.Appearance.Host = access
	}
	if m.Models != nil {
		m.Models.OnConfigure = func(namespace string) {
			m.Store.Select(namespace)
			// This configures a provider within Models, preserving its page owner.
			s.mu.Lock()
			s.settingsEmbeddedSection = ""
			s.settingsModelsActive = true
			s.mu.Unlock()
		}
	}
	if m.About != nil {
		m.About.HomePath = paths.DshHome()
		m.About.OnRevealHome = func() error { return desktopcomponents.OpenInFileManager(paths.DshHome()) }
		s.mu.Lock()
		releases := s.nativeReleases
		checking := s.updateCheckPending
		win := s.win
		s.mu.Unlock()
		m.About.UpdateBusy = releases.busy || checking
		m.About.UpdateStatus = strings.TrimSpace(releases.status + " " + releases.err)
		m.About.OnInstallUpdate = func() { s.downloadNativeRelease(win, "", false) }
	}
	var cfg config.Config
	if s.cfgStore != nil {
		cfg = s.cfgStore.Get()
	}
	if m.Interface != nil {
		m.Interface.Host = access
		m.Interface.DshbotEnabled = &cfg.DshbotEnabled
		m.Interface.RemoteWorkspaceEnabled = &cfg.RemoteWorkspaceEnabled
		m.Interface.OnSetDshbot = func(v bool) { s.setAssistantFlag(nil, &v) }
		m.Interface.OnSetRemoteWorkspace = s.setRemoteWorkspacePreference
	}
	s.mu.Lock()
	ep, ready := s.bootStatus.Endpoint, s.bootStatus.State == harness.Ready
	petSaveErr := s.desktopSaveErr
	s.mu.Unlock()
	petSettings := &settings.PetSettingsAccess{
		Err: petSaveErr,
		Store: m.Store, AssistantEnabled: cfg.WhaleAssistantEnabled, AssistantAvailable: s.cfgStore != nil,
		OnSetAssistant: func(v bool) { s.setAssistantFlag(&v, nil) },
		RenderAssistant: func(c *ui.Context) {
			assistant.WhaleSettingsSection(c, assistant.Model{Mode: assiststore.ModeWhale,
				HarnessReady: ready, WhaleEnabled: cfg.WhaleAssistantEnabled, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
				WhaleSettings: s.assistantWhaleSettingsModel(ep)})
		},
	}
	// The dsh-whale catalog is Host data: without a ready Host every field and
	// callback stays nil and the page renders exactly as before.
	petSettings.Whale, petSettings.OnLoadWhale, petSettings.OnUpdateWhale,
		petSettings.OnReplaceMemory, petSettings.OnClearMemory, petSettings.OnToggleSkill =
		s.whalePetAccess(ep, ready)
	m.PetSettings = petSettings
}

func (s *mainState) setRemoteWorkspacePreference(v bool) {
	if s.cfgStore == nil {
		return
	}
	if err := s.cfgStore.Update(func(c *config.Config) { c.RemoteWorkspaceEnabled = v }); err != nil {
		s.mu.Lock()
		s.desktopSaveErr = err.Error()
		s.mu.Unlock()
		return
	}
	s.syncHarnessFromConfig()
	s.mu.Lock()
	sup := s.sup
	win := s.win
	s.mu.Unlock()
	if sup != nil {
		s.restartUnlessImport(sup)
	}
	if win != nil {
		win.Update(func() {})
	}
}
