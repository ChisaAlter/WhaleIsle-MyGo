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
	cfgStore := s.cfgStore
	if cfgStore == nil {
		s.mu.Unlock()
		return !next, errors.New(i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	switch key {
	case "archive":
		s.desktopDraft.showArchivedList = next
	case "tabs":
		s.desktopDraft.showViewTabs = next
	case "tools":
		s.desktopDraft.autoAllowTools = next
	}
	interactions := s.interactions
	win := s.win
	s.mu.Unlock()

	s.enqueueConfigUpdate(cfgStore, func(c *config.Config) {
		switch key {
		case "archive":
			c.HideArchivedList = !next
		case "tabs":
			c.HideChatViewTabs = !next
		case "tools":
			c.AutoAllowTools = next
		}
	})
	if key == "tools" && interactions != nil {
		interactions.SetAutoAllow(next)
	}
	if win != nil {
		win.Invalidate()
	}
	return next, nil
}
