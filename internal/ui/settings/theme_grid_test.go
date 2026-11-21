package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func themeGridFixture(t *testing.T) (*theme.Editor, *ui.Tester) {
	t.Helper()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	editor := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
	colors := theme.Resolve(theme.IDDeepseekLight).Light
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() {
			themeLibraryPane(c, editor, colors, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 760, 900)
	return editor, tt
}

// TestThemeGridShowsEveryFamilyWithBothHalves covers the appearance grid: each
// theme family paints one card whose light and dark halves are separately
// clickable, instead of a pair of dropdowns.
func TestThemeGridShowsEveryFamilyWithBothHalves(t *testing.T) {
	editor, tt := themeGridFixture(t)
	families := editor.Draft.Families()
	if len(families) == 0 {
		t.Fatal("fixture has no theme families")
	}
	for _, family := range families {
		if !tt.HasText(family.LocalizedName()) {
			t.Fatalf("theme grid dropped %q: %v", family.LocalizedName(), tt.Texts())
		}
		if !tt.HasText(family.LocalizedName()+" light") || !tt.HasText(family.LocalizedName()+" dark") {
			t.Fatalf("theme card %q lost a preview half", family.LocalizedName())
		}
	}
	if !tt.HasText(i18n.T(i18n.KeySurfacesSettingsReset)) && !tt.HasText("Create theme") {
		t.Fatalf("theme header lost its actions: %v", tt.Texts())
	}
}

// TestThemeHalfPicksThatScheme keeps the halves meaningful: the light preview
// selects the light scheme and the dark preview the dark one.
func TestThemeHalfPicksThatScheme(t *testing.T) {
	editor, tt := themeGridFixture(t)
	families := editor.Draft.Families()
	target := families[len(families)-1].ID
	if err := tt.Click(families[len(families)-1].LocalizedName() + " dark"); err != nil {
		t.Fatalf("dark preview is not clickable: %v", err)
	}
	if editor.Draft.Dark != target {
		t.Fatalf("dark preview selected %q, want %q", editor.Draft.Dark, target)
	}
	if err := tt.Click(families[0].LocalizedName() + " light"); err != nil {
		t.Fatalf("light preview is not clickable: %v", err)
	}
	if editor.Draft.Light != families[0].ID {
		t.Fatalf("light preview selected %q, want %q", editor.Draft.Light, families[0].ID)
	}
}

func TestThemeFamilyTitleUsesOneLocale(t *testing.T) {
	i18n.SetLocale("zh")
	if got := (theme.Family{ID: "midnight", Name: "午夜"}).LocalizedName(); got != "午夜" {
		t.Fatalf("zh title = %q", got)
	}
	if got := (theme.Family{Name: "午夜 / Midnight"}).LocalizedName(); got != "午夜" {
		t.Fatalf("zh legacy title = %q", got)
	}
	i18n.SetLocale(i18n.LocaleEN)
	if got := (theme.Family{ID: "midnight", Name: "午夜"}).LocalizedName(); got != "Midnight" {
		t.Fatalf("en title = %q", got)
	}
	if got := (theme.Family{Name: "午夜 / Midnight"}).LocalizedName(); got != "Midnight" {
		t.Fatalf("en legacy title = %q", got)
	}
	if got := (theme.Family{Name: "DeepSeek"}).LocalizedName(); got != "DeepSeek" {
		t.Fatalf("plain title = %q", got)
	}
	i18n.SetLocale(i18n.DefaultLocale)
}
