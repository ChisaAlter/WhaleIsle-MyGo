package main

import (
	"testing"

	"whaleisle/internal/config"
)

// TestStartupTheme pins the launch resolution: a configured scheme wins until
// the user makes an explicit appearance choice, then the persisted preference
// wins even when config later changes.
func TestStartupTheme(t *testing.T) {
	for _, tc := range []struct {
		name         string
		cfgTheme     string
		explicit     bool
		hostScheme   string
		hostAccepted bool
		want         string
	}{
		{"config dark with no host preference", "deepseek.dark", false, "system", true, "deepseek.dark"},
		{"config light with no host preference", "deepseek.light", false, "system", true, "deepseek.light"},
		{"explicit host light beats config dark", "deepseek.dark", false, "light", true, "deepseek.light"},
		{"explicit host dark beats config light", "deepseek.light", false, "dark", true, "deepseek.dark"},
		{"user toggled light, later config dark", "deepseek.dark", true, "light", true, "deepseek.light"},
		{"user chose system explicitly", "deepseek.dark", true, "system", true, "deepseek"},
		{"host not accepted uses config", "deepseek.dark", false, "system", false, "deepseek.dark"},
		{"system config follows the OS", "deepseek", false, "system", true, "deepseek"},
		{"empty theme falls back to default", "", false, "", false, "deepseek"},
		{"host scheme is trimmed", "deepseek.dark", false, " DARK ", true, "deepseek.dark"},
	} {
		got := startupTheme(config.Config{Theme: tc.cfgTheme, AppearanceExplicit: tc.explicit}, tc.hostScheme, tc.hostAccepted)
		if got != tc.want {
			t.Fatalf("%s: startupTheme(%q, %q, %v) = %q, want %q", tc.name, tc.cfgTheme, tc.hostScheme, tc.hostAccepted, got, tc.want)
		}
	}
}
