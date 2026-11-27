package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSettingsHoverFocusZeroShiftSweep(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	host := bodyFixtureHost()
	editor := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	minOn, minSuggest := false, true
	glassOn := true
	themeChoice := "system"
	wpPrefs := wallpapers.Preferences{Glass: 80, Blur: 24, Terminal: 90}
	wpEditor := &WallpaperEditor{
		Prefs: wpPrefs,
		Saved: wpPrefs,
		Save:  func(p wallpapers.Preferences) error { wpPrefs = p; return nil },
	}
	closeToTray := false
	autoStart := false
	locale := "zh"
	selectedPreset := ""

	base := fullRailModel()
	base.Desktop = &DesktopAccess{
		Host:             host,
		Preferences:      editor,
		AutoStartDesktop: &autoStart,
		Fields: DesktopFields{
			Locale:      &locale,
			CloseToTray: &closeToTray,
		},
		OnSetLocale:           func(v string) (string, error) { return v, nil },
		OnSetCloseToTray:      func(v bool) (bool, error) { return v, nil },
		OnSetAutoStartDesktop: func(v bool) (bool, error) { return v, nil },
	}
	base.Appearance = &AppearanceAccess{
		Host: host,
		Minimal: &MinimalModeAccess{
			Enabled:             &minOn,
			Suggestions:         &minSuggest,
			OnToggle:            func(bool) error { return nil },
			OnToggleSuggestions: func(bool) error { return nil },
		},
		Glass: &GlassThemeAccess{
			Enabled:  &glassOn,
			OnToggle: func(bool) error { return nil },
		},
		Background:  wpEditor,
		Preferences: editor,
		Theme:       &themeChoice,
		OnSetTheme:  func(string) error { return nil },
	}
	base.AgentPresets = &AgentPresetsAccess{
		Selected: &selectedPreset,
		Reader:   &AgentPresetReaderAccess{OnSelect: func(uint64, string) {}},
	}
	base.About = &AboutAccess{
		AppName:            "Whale Isle",
		Version:            "dev",
		OnOpenUpdateCenter: func() {},
		OnCheckUpdates:     func() {},
	}

	for _, sec := range []SectionID{SectionGeneral, SectionAppearance, SectionAgentPresets, SectionAbout} {
		t.Run(string(sec), func(t *testing.T) {
			cur := base
			cur.ActiveSection = sec
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, cur)
			}, 1000, 760)
			tt.Frame()
			tt.Frame()
			ui.AssertHoverFocusZeroShift(t, tt)
		})
	}
}
