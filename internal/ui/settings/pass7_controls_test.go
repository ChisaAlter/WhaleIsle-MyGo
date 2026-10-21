package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// TestConfigFileActionIsLiveWhenWired covers A1-1: with OnOpenConfig wired the
// rail action is a real button, and without it the same button explains itself
// instead of promising an unimplemented action.
func TestConfigFileActionIsLiveWhenWired(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	opened := 0
	m := Model{ThemeID: "deepseek.light", OnOpenConfig: func() { opened++ }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			header(c, m, store.ViewState{}, c.Theme().Text, c.Theme().TextMuted, c.Theme().Border)
		})
	}, 800, 160)
	label := i18n.T(i18n.KeySurfacesSettingsOpenConfig)
	if err := tt.Click(label); err != nil {
		t.Fatalf("config action is not clickable although it is wired: %v", err)
	}
	if opened != 1 {
		t.Fatalf("config action ran %d times, want 1", opened)
	}
}

// TestAboutVersionIsReadOnlyText covers A2-6: the installed version is data, so
// it no longer paints a one-option disabled picker.
func TestAboutVersionIsReadOnlyText(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	a := &AboutAccess{AppName: "Whale Isle", Version: "0.3.9"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			aboutPane(c, Model{About: a, ThemeID: "deepseek.light"}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 800, 700)
	if !tt.HasText("0.3.9") {
		t.Fatalf("version is not painted as text: %v", tt.Texts())
	}
	rect, ok := tt.Find(i18n.T(i18n.KeySettingsAboutCopyVersion))
	if !ok {
		t.Fatalf("version row lost its label: %v", tt.Texts())
	}
	if rect.H != 28 {
		t.Fatalf("version row is %v tall, want the 28 DIP text row", rect.H)
	}
}
