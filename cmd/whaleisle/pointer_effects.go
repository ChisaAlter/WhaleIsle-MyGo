package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// configurePointerEffects installs the visible pointer decoration and button
// sheen for the current appearance settings. chrome.ApplyCursorEffects owns the
// clamping, palette validation, accent fallback and reduced-motion rules so the
// shell cannot publish values the renderer would reject.
func (s *mainState) configurePointerEffects(c *ui.Context) {
	g := s.wallpaperEditor()
	if g == nil {
		return
	}
	systemDark := mygo.Theme.IsDark()
	colors := theme.ResolveWithSystem(s.themeID, systemDark).Active(systemDark)
	// Minimal mode pauses the pointer decoration through the same projection the
	// backdrop uses; the stored preference is what comes back when it is off.
	chrome.ApplyCursorEffects(c, renderpolicy.Background(g.EffectivePreferences()), chrome.Color(colors.Accent))
}
