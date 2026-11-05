package main

import (
	"errors"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/i18n"
)

// General's language selector owns only locale, never the diagnostic draft.
func (s *mainState) setLocalePreference(locale string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfgStore == nil {
		return i18n.Locale(), errors.New(i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	if err := s.cfgStore.Update(func(c *config.Config) { c.Locale = locale }); err != nil {
		return s.cfgStore.Get().Locale, err
	}
	accepted := s.cfgStore.Get().Locale
	s.desktopDraft.locale = accepted
	i18n.SetLocale(accepted)
	return accepted, nil
}
