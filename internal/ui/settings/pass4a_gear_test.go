package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func interfacePaneTester(t *testing.T, a *InterfaceAccess, width, height int) *ui.Tester {
	t.Helper()
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			interfacePane(c, Model{Interface: a, ThemeID: theme.IDDeepseekLight}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, width, height)
}

func interfaceGearFixture() *InterfaceAccess {
	h := bodyFixtureHost()
	enabled := true
	return &InterfaceAccess{
		Host:             h,
		ShowViewTabs:     &enabled,
		ShowArchivedList: &enabled,
		OnSave:           func() {},
	}
}

// TestInterfaceGearOnlyOnTheGlowRow pins the rule from the reference: exactly one
// row carries the gear, and it is the composer beam (thinking glow) row.
func TestInterfaceGearOnlyOnTheGlowRow(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	tt := interfacePaneTester(t, interfaceGearFixture(), 900, 1200)
	gear := i18n.T(KeyPass4AGlowSettings)
	count := 0
	for _, text := range tt.Texts() {
		if text == gear {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("gear label appears %d times, want 1 (%s)", count, gear)
	}
	rect, ok := tt.Find(gear)
	if !ok {
		t.Fatalf("gear is not reachable: %v", tt.Texts())
	}
	if rect.W != 24 || rect.H != 24 {
		t.Fatalf("gear hit area is %v×%v, want 24×24", rect.W, rect.H)
	}
}

// TestInterfaceGearOpensTheEffectDialog keeps the gear wired: clicking it shows
// the effect dialog exactly the way the row's Configure button does.
func TestInterfaceGearOpensTheEffectDialog(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	tt := interfacePaneTester(t, interfaceGearFixture(), 900, 1200)
	dialogTitle := interfaceEffectTitle()
	if tt.HasText(dialogTitle) {
		t.Fatalf("effect dialog is open before any click: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(KeyPass4AGlowSettings)); err != nil {
		t.Fatalf("gear is not clickable: %v", err)
	}
	tt.Frame()
	if !tt.HasText(dialogTitle) {
		t.Fatalf("gear did not open the effect dialog (looking for %q)", dialogTitle)
	}
}

// interfaceEffectTitle is a line only the effect dialog paints (the row title is
// the same words, so the dialog needs its own marker).
func interfaceEffectTitle() string {
	return providerCopy("个人预设", "Personal presets")
}
