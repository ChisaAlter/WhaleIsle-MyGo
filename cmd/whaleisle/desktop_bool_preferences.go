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
// Disk I/O runs without holding s.mu, after flushing any pending debounced writes
// so recent async edits (such as locale or theme) are never rolled back.
func (s *mainState) setDesktopBoolPreference(preference desktopBoolPreference, next bool) (bool, error) {
	s.mu.Lock()
	store := s.cfgStore
	if store == nil {
		accepted := s.desktopDraft.autoStartDesktop
		if preference == desktopCloseToTray {
			accepted = s.closeToTray
		}
		s.mu.Unlock()
		return accepted, errors.New(i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	s.mu.Unlock()

	if err := s.flushExistingConfigWriter(store); err != nil {
		return desktopBoolValue(s.effectiveConfigFor(store), preference), err
	}
	if err := store.Update(func(c *config.Config) {
		if preference == desktopCloseToTray {
			c.CloseToTray = next
		} else {
			c.AutoStartDesktop = next
		}
	}); err != nil {
		return desktopBoolValue(s.effectiveConfigFor(store), preference), err
	}
	accepted := desktopBoolValue(s.effectiveConfigFor(store), preference)
	if accepted != next {
		return accepted, errors.New(i18n.T(i18n.KeySurfacesSettingsSaveFailed))
	}
	s.mu.Lock()
	win := s.win
	if s.cfgStore == store {
		if preference == desktopCloseToTray {
			s.closeToTray = accepted
			s.desktopDraft.closeToTray = accepted
		} else {
			s.desktopDraft.autoStartDesktop = accepted
		}
	}
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
	return accepted, nil
}
