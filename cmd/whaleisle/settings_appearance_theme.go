package main

import (
	"fmt"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/i18n"
)

// Selecting a color scheme commits only that field, preserving other open
// appearance/desktop drafts until their own explicit save.
func (s *mainState) setAppearanceTheme(id string) error {
	s.mu.Lock()
	cfg := s.cfgStore
	win := s.win
	host, hostAccepted := s.hostAppearance, s.hostAppearanceAccepted
	s.mu.Unlock()
	if hostAccepted {
		if host == nil {
			return fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostAppearanceUnavailable))
		}
		if err := host.SaveScheme(id); err != nil {
			return err
		}
		// Record the choice so a later config edit cannot take the scheme back.
		if cfg != nil {
			if err := cfg.Update(func(c *config.Config) {
				c.Theme = id
				c.AppearanceExplicit = true
			}); err != nil {
				return err
			}
		}
		s.mu.Lock()
		s.themeID, s.desktopDraft.theme = id, id
		s.mu.Unlock()
		applyAppearance(id)
		if win != nil {
			win.Invalidate()
		}
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("%s", i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	if err := cfg.Update(func(c *config.Config) {
		c.Theme = id
		c.AppearanceExplicit = true
	}); err != nil {
		return err
	}
	s.mu.Lock()
	s.themeID = id
	s.desktopDraft.theme = id
	s.mu.Unlock()
	applyAppearance(id)
	if win != nil {
		win.Update(func() {})
	}
	return nil
}
