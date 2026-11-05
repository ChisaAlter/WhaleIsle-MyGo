package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
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
	chrome.ApplyCursorEffects(c, g.EffectivePreferences(), chrome.Color(colors.Accent))
}
