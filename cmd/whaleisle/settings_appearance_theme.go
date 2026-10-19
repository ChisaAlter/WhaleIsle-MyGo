package main

import (
	"fmt"
	"whaleisle/internal/config"
)

// Selecting a color scheme commits only that field, preserving other open
// appearance/desktop drafts until their own explicit save.
func (s *mainState) setAppearanceTheme(id string) error {
	s.mu.Lock()
	cfg := s.cfgStore
	win := s.win
	s.mu.Unlock()
	if cfg == nil {
		return fmt.Errorf("config store is not ready")
	}
	if err := cfg.Update(func(c *config.Config) { c.Theme = id }); err != nil {
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
