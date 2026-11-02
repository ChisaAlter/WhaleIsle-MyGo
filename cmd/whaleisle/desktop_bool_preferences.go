package main

import (
	"errors"

	"whaleisle/internal/config"
	"whaleisle/internal/ui/i18n"
)

type desktopBoolPreference uint8

const (
	desktopCloseToTray desktopBoolPreference = iota
	desktopAutoStart
)

func desktopBoolValue(cfg config.Config, preference desktopBoolPreference) bool {
	if preference == desktopCloseToTray {
		return cfg.CloseToTray
	}
	return cfg.AutoStartDesktop
}

func (s *mainState) setCloseToTrayPreference(next bool) (bool, error) {
	return s.setDesktopBoolPreference(desktopCloseToTray, next)
}

func (s *mainState) setAutoStartDesktopPreference(next bool) (bool, error) {
	return s.setDesktopBoolPreference(desktopAutoStart, next)
}

// These General selectors persist one field immediately. They must not parse or
// save any separate desktop form draft, restart Host, or change OS login items.
// The state lock orders the commit/readback/publication against the local Save
// operation. Store.Update retains its existing atomic write and conflict checks.
func (s *mainState) setDesktopBoolPreference(preference desktopBoolPreference, next bool) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfgStore == nil {
		accepted := s.desktopDraft.autoStartDesktop
		if preference == desktopCloseToTray {
			accepted = s.closeToTray
		}
		return accepted, errors.New(i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	store := s.cfgStore
	if err := store.Update(func(c *config.Config) {
		if preference == desktopCloseToTray {
			c.CloseToTray = next
		} else {
			c.AutoStartDesktop = next
		}
	}); err != nil {
		return desktopBoolValue(store.Get(), preference), err
	}
	accepted := desktopBoolValue(store.Get(), preference)
	if accepted != next {
		return accepted, errors.New(i18n.T(i18n.KeySurfacesSettingsSaveFailed))
	}
	if preference == desktopCloseToTray {
		s.closeToTray = accepted
		s.desktopDraft.closeToTray = accepted
	} else {
		s.desktopDraft.autoStartDesktop = accepted
	}
	return accepted, nil
}
