package main

import (
	"context"

	"github.com/egoist/mygo/ui"
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
		s.settingsConversationActive || s.settingsFeaturesActive ||
		s.settingsModelsActive || s.settingsMCPActive || s.settingsPluginsActive ||
		s.settingsSkillsActive || s.settingsAgentPresetsActive {
		return false
	}
	ns := s.settings.SelectedNamespace()
	return ns == "pet" || ns == "live2d"
}

func (s *mainState) wireSettingsBodies(m *settings.Model) {
	if m.Store == nil {
		return
	}
	if m.AgentPresetsActive && m.AgentPresets != nil {
		m.AgentPresets.ACP = s.acpSettingsAccess()
	}
	m.OnOpenConfig = nil
	documentState := m.Store.Snapshot().Value
	if documentState.ClientReady && documentState.Described && documentState.HasDocument {
		m.OnOpenConfig = func() { m.Store.OpenSettingsDocument(context.Background()) }
	}
	if m.Account != nil {
		m.Account.OnOpenInfo = func() { s.openAccountInfo(m.Account.UsageURL) }
	}
	var access *settings.BodySettingsAccess
	if m.DesktopActive || m.AppearanceActive || m.InterfaceActive || m.ConversationActive {
		snapshot, owner, write := m.Store.BindDescribedPreferences()
		access = &settings.BodySettingsAccess{Snapshot: snapshot, Owner: owner, OnChange: func(ns, key, value string) {
			write(context.Background(), ns, map[string]string{key: value})
		}, OnPatch: func(ns string, values map[string]string) {
			write(context.Background(), ns, values)
		}}
	}
	if m.DesktopActive && m.Desktop != nil {
		m.Desktop.OnSetLocale = s.setLocalePreference
		m.Desktop.OnSetRestartPreference = s.setHarnessRestartPreference
		m.Desktop.Host = access
		m.Desktop.Shortcuts = s.pageShortcutEditor()
		m.Desktop.Preferences = s.themeFontEditor(true)
		m.Desktop.Version = s.harnessVersionSnapshot()
		m.Desktop.UpdateBusy = s.updaterCenter().FrameSnapshot().Busy
		m.Desktop.AutoStartDesktop = &s.desktopDraft.autoStartDesktop
	}
	if m.AppearanceActive && m.Appearance != nil {
		m.Appearance.Host = access
	}
	if m.ConversationActive && m.Conversation != nil {
		m.Conversation.Host = access
	}
	if m.FeaturesActive && m.Features != nil {
		var cfg config.Config
		if s.cfgStore != nil {
			cfg = s.cfgStore.Get()
		}
		m.Features.Project = s.projectPreferenceAccess()
		m.Features.Assistant = s.featurePreferenceAccess(whaleAssistantFeature)
		m.Features.RemoteWorkspace = s.featurePreferenceAccess(remoteWorkspaceFeature)
		m.Features.AssistantEnabled = &cfg.WhaleAssistantEnabled
		m.Features.OnSetAssistant = func(v bool) { s.setAssistantFlag(&v, nil) }
		m.Features.RemoteWorkspaceEnabled = &cfg.RemoteWorkspaceEnabled
		m.Features.OnSetRemoteWorkspace = s.setRemoteWorkspacePreference
	}
	if m.ModelsActive && m.Models != nil {
		m.Models.OnConfigure = func(namespace string) {
			m.Store.Select(namespace)
			// This configures a provider within Models, preserving its page owner.
			s.mu.Lock()
			s.settingsEmbeddedSection = ""
			s.settingsModelsActive = true
			s.mu.Unlock()
		}
	}
	if m.AboutActive && m.About != nil {
		u := s.updaterCenter()
		s.mu.Lock()
		aboutWin := s.win
		s.mu.Unlock()
		aboutRef := updaterWindow(aboutWin)
		aboutSnap := u.AboutSnapshot(aboutRef)
		m.About.HomePath = paths.DshHome()
		m.About.OnRevealHome = func() error { return desktopcomponents.OpenInFileManager(paths.DshHome()) }
		m.About.UpdateBusy = aboutSnap.UpdateBusy
		m.About.UpdateStatus = aboutSnap.UpdateStatus
		m.About.UpdatePhase = aboutSnap.UpdatePhase
		m.About.UpdateVersion = aboutSnap.UpdateVersion
		m.About.UpdatePercent = aboutSnap.UpdatePercent
		m.About.OnDownloadUpdate = func() { u.OpenCenter(aboutRef) }
		m.About.OnResumeUpdate = func() { u.OpenCenter(aboutRef) }
		if aboutSnap.ActiveTag != "" {
			m.About.OnResumeUpdate = func() { u.Resume(aboutRef) }
		}
		if aboutSnap.StagedPath != "" || m.About.UpdatePhase == "ready" {
			m.About.OnInstallUpdate = func() { u.StartStaged(aboutRef) }
		} else {
			m.About.OnInstallUpdate = func() { u.OpenCenter(aboutRef) }
		}
		m.About.OnOpenUpdateCenter = func() { u.OpenCenter(aboutRef) }
	}
	if m.InterfaceActive && m.Interface != nil {
		var cfg config.Config
		if s.cfgStore != nil {
			cfg = s.cfgStore.Get()
		}
		m.Interface.Host = access
		m.Interface.Project = s.projectPreferenceAccess()
		m.Interface.DshbotEnabled = &cfg.DshbotEnabled
		m.Interface.RemoteWorkspaceEnabled = &cfg.RemoteWorkspaceEnabled
		m.Interface.OnSetDshbot = func(v bool) { s.setAssistantFlag(nil, &v) }
		m.Interface.OnSetRemoteWorkspace = s.setRemoteWorkspacePreference
	}
	s.mu.Lock()
	ep, ready, petActive := s.bootStatus.Endpoint, s.bootStatus.State == harness.Ready, s.settingsPetActiveLocked()
	petSaveErr := s.desktopSaveErr
	desktopErr := ""
	if s.desktopPet != nil {
		desktopErr = s.desktopPet.Error
	}
	s.mu.Unlock()
	if petActive {
		var cfg config.Config
		if s.cfgStore != nil {
			cfg = s.cfgStore.Get()
		}
		petSettings := &settings.PetSettingsAccess{
			AssistantPreference: s.featurePreferenceAccess(whaleAssistantFeature),
			Err:                 petSaveErr,
			Store:               m.Store, AssistantEnabled: cfg.WhaleAssistantEnabled, AssistantAvailable: s.cfgStore != nil, DesktopUnavailable: isolatedDesktopRun(), DesktopError: desktopErr,
			OnSetAssistant: func(v bool) { s.setAssistantFlag(&v, nil) },
			RenderAssistant: func(c *ui.Context) {
				feature := s.featurePreferenceAccess(whaleAssistantFeature)
				featureBusy := feature != nil && feature.State.Busy
				assistant.WhaleSettingsSection(c, assistant.Model{Mode: assiststore.ModeWhale, Busy: featureBusy,
					HarnessReady: ready, WhaleEnabled: cfg.WhaleAssistantEnabled, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
					WhaleSettings: s.assistantWhaleSettingsModel(ep)})
			},
		}
		m.PetSettings = petSettings
	}
}

func (s *mainState) setRemoteWorkspacePreference(v bool) {
	s.setFeaturePreference(remoteWorkspaceFeature, v)
}
