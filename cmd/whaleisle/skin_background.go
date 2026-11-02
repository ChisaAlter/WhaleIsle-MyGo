package main

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

// skinBackdrop is a render-only projection. Codex's flatChrome branch ignores
// saved wallpaper, gradient and transparent/glass settings without erasing them.
func skinBackdrop(skin, path string, prefs wallpapers.Preferences) (string, wallpapers.Preferences) {
	if skin == theme.SkinCodex {
		path = ""
		prefs.Effect, prefs.EffectExplicit = "none", true
		prefs.Glass, prefs.Terminal = 100, 100
		prefs.Transparent = false
	}
	return path, prefs
}

func (s *mainState) activeSkin() string {
	s.mu.Lock()
	id := s.themeID
	s.mu.Unlock()
	// Theme.IsDark marshals to the live app event loop. Pure render-policy
	// tests have no app loop; pinned schemes remain independent of this value.
	systemDark := false
	if mygo.App.IsReady() {
		systemDark = mygo.Theme.IsDark()
	}
	return theme.ResolvedSkin(id, systemDark)
}
