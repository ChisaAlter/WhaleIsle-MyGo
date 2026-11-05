package main

import (
	"errors"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) setShowArchivedListPreference(next bool) (bool, error) {
	return s.setInterfaceBoolPreference("archive", next)
}
func (s *mainState) setLocalViewTabsPreference(next bool) (bool, error) {
	return s.setInterfaceBoolPreference("tabs", next)
}
func (s *mainState) setAutoAllowToolsPreference(next bool) (bool, error) {
	return s.setInterfaceBoolPreference("tools", next)
}

func (s *mainState) setInterfaceBoolPreference(key string, next bool) (bool, error) {
	s.mu.Lock()
	read := func(c config.Config) bool {
		switch key {
		case "archive":
			return !c.HideArchivedList
		case "tabs":
			return !c.HideChatViewTabs
		default:
			return c.AutoAllowTools
		}
	}
	if s.cfgStore == nil {
		s.mu.Unlock()
		return !next, errors.New(i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	err := s.cfgStore.Update(func(c *config.Config) {
		switch key {
		case "archive":
			c.HideArchivedList = !next
		case "tabs":
			c.HideChatViewTabs = !next
		case "tools":
			c.AutoAllowTools = next
		}
	})
	accepted := read(s.cfgStore.Get())
	if err != nil {
		s.mu.Unlock()
		return accepted, err
	}
	if accepted != next {
		s.mu.Unlock()
		return accepted, errors.New(i18n.T(i18n.KeySurfacesSettingsSaveFailed))
	}
	switch key {
	case "archive":
		s.desktopDraft.showArchivedList = accepted
	case "tabs":
		s.desktopDraft.showViewTabs = accepted
	case "tools":
		s.desktopDraft.autoAllowTools = accepted
	}
	interactions := s.interactions
	s.mu.Unlock()
	if key == "tools" && interactions != nil {
		interactions.SetAutoAllow(accepted)
	}
	return accepted, nil
}
