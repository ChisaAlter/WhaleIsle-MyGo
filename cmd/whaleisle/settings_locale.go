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
	accepted := i18n.Language(locale)
	if accepted != "en" && accepted != "zh" {
		accepted = i18n.Language(s.effectiveConfigFor(cfgStore).Locale)
		if accepted != "en" && accepted != "zh" {
			accepted = "zh"
		}
	}
	s.desktopDraft.locale = accepted
	win := s.win
	s.mu.Unlock()
	i18n.SetLocale(accepted)
	s.enqueueConfigUpdate(cfgStore, func(c *config.Config) { c.Locale = accepted })
	if win != nil {
		win.Invalidate()
	}
	return accepted, nil
}
