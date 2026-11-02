package main

import (
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/settings"
)

func (s *mainState) conversationViewTabsHidden(snapshot settingsstore.ViewState) bool {
	if on, accepted := settings.InterfaceViewTabsAccepted(&settings.BodySettingsAccess{Snapshot: snapshot}); accepted {
		return !on
	}
	if s.cfgStore != nil {
		return s.cfgStore.Get().HideChatViewTabs
	}
	return false
}

// saveInterfacePreferences saves local preferences without writing a retained
// view-tabs mirror over the current Host's independently accepted preference.
func (s *mainState) saveInterfacePreferences() {
	s.mu.Lock()
	if s.settings != nil {
		snapshot, _, _ := s.settings.BindDescribedPreferences()
		access := &settings.BodySettingsAccess{Snapshot: snapshot}
		fallback := settings.InterfaceViewTabsFallbackAllowed(access)
		if on, accepted := settings.InterfaceViewTabsAccepted(access); accepted {
			s.desktopDraft.showViewTabs = on
		} else if !fallback && s.cfgStore != nil {
			// An unresolved or unsupported Host cannot authorize the hidden draft.
			s.desktopDraft.showViewTabs = !s.cfgStore.Get().HideChatViewTabs
		}
	}
	s.saveDesktopConfig()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
