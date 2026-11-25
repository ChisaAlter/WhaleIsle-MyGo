package main

import (
	"errors"

	"whaleisle/internal/config"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/settings"
)

// applyGlassTheme puts the optional wallpaper-and-glass theme in force for
// every window: with it off, the backdrop painter configures no wallpaper,
// gradient or glass sampling at all.
func applyGlassTheme(on bool) { renderpolicy.SetGlass(on) }

var errGlassThemeUnavailable = errors.New("glass theme: the settings store is not open yet")

// migrateGlassTheme stores the effective switch once, so a profile from
// before the switch keeps the decision its own wallpaper choice implied.
func migrateGlassTheme(store *config.Store) bool {
	cfg := store.Get()
	on := cfg.GlassThemeOn()
	if cfg.GlassTheme == nil {
		_ = store.Update(func(c *config.Config) { v := on; c.GlassTheme = &v })
	}
	return on
}

// setGlassTheme applies the switch optimistically in memory and persists it
// asynchronously on the single-writer debounced config queue.
func (s *mainState) setGlassTheme(on bool) error {
	if s.cfgStore == nil {
		return errGlassThemeUnavailable
	}
	applyGlassTheme(on)
	s.enqueueConfigUpdate(s.cfgStore, func(c *config.Config) { v := on; c.GlassTheme = &v })
	s.invalidateAppearance()
	return nil
}

// glassThemeAccess is the appearance page's glass-theme switch.
func (s *mainState) glassThemeAccess() *settings.GlassThemeAccess {
	if s.cfgStore == nil {
		return nil
	}
	enabled := s.effectiveConfigFor(s.cfgStore).GlassThemeOn()
	return &settings.GlassThemeAccess{Enabled: &enabled, OnToggle: s.setGlassTheme}
}
