package main

import (
	"errors"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/i18n"
)

// General's language selector owns only locale, never the diagnostic draft.
// The locale takes effect optimistically in memory and persists asynchronously
// so switching languages never stalls the UI thread on disk fsync.
func (s *mainState) setLocalePreference(locale string) (string, error) {
	s.mu.Lock()
	cfgStore := s.cfgStore
	if cfgStore == nil {
		s.mu.Unlock()
		return i18n.Locale(), errors.New(i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	accepted := locale
	if accepted != i18n.LocaleEN && accepted != i18n.LocaleZH {
		accepted = s.effectiveConfigFor(cfgStore).Locale
	}
	s.desktopDraft.locale = accepted
	s.mu.Unlock()
	i18n.SetLocale(accepted)
	s.enqueueConfigUpdate(cfgStore, func(c *config.Config) { c.Locale = accepted })
	return accepted, nil
}
