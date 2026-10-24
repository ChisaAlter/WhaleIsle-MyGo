package main

import (
	"strings"

	"whaleisle/internal/config"
	"whaleisle/internal/ui/theme"
)

// schemeThemeID maps a Host appearance preference to the theme id this shell
// renders. "system" keeps the deepseek system scheme, which follows the OS.
func schemeThemeID(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "light":
		return theme.IDDeepseekLight
	case "dark":
		return theme.IDDeepseekDark
	default:
		return "deepseek"
	}
}

// startupTheme resolves the window theme at launch. The Host always reports a
// scheme (its default is "system"), so the Host alone cannot tell an explicit
// user choice from its default: before the user ever chooses an appearance the
// configured theme stays authoritative, and after that choice (a persisted Host
// scheme other than system, or the explicit marker in config) the preference
// wins and keeps winning when config later changes.
func startupTheme(cfg config.Config, hostScheme string, hostAccepted bool) string {
	if hostAccepted && (cfg.AppearanceExplicit || strings.ToLower(strings.TrimSpace(hostScheme)) != "system") {
		return schemeThemeID(hostScheme)
	}
	if cfg.Theme != "" {
		return cfg.Theme
	}
	return config.DefaultTheme
}
