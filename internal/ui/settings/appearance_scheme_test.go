package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestAppearanceOffersThreeColorModes pins the appearance page's first block: the
// three scheme cards, and a click that reports the matching theme id.
func TestAppearanceOffersThreeColorModes(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	h := bodyFixtureHost()
	scheme := "system"
	picked := ""
	prefs := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	background := NewWallpaperEditor(wallpapers.Defaults(), "", func(wallpapers.Preferences) error { return nil }, nil, nil)
	appearance := &AppearanceAccess{
		Host:        h,
		Preferences: prefs,
		Background:  background,
		Theme:       &scheme,
		OnSetTheme:  func(next string) error { picked = next; return nil },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		appearancePane(c, Model{Appearance: appearance, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1400)
	for _, label := range []string{"Light", "Dark", "System"} {
		if !tt.HasText(label) {
			t.Fatalf("color scheme card %q missing: %v", label, tt.Texts())
		}
	}
	if err := tt.Click("Dark"); err != nil {
		t.Fatalf("dark scheme card is not clickable: %v", err)
	}
	if picked != "deepseek.dark" {
		t.Fatalf("dark card selected %q", picked)
	}
}
