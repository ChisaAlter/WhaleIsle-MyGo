package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestContributedSectionIsSelectableInsideExistingSettingsRail(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	selected := 0
	m := Model{DesktopActive: true, Desktop: &DesktopAccess{}}
	m.OnSelectDesktop = func() { m.DesktopActive, m.ActiveSection, m.SectionContent = true, "", nil }
	m.ContributedSections = []ContributedSection{{ID: "plugin/native-settings/owner-1", Label: "Native plugin settings", OnSelect: func() {
		selected++
		m.DesktopActive, m.ActiveSection = false, "plugin/native-settings/owner-1"
		m.SectionContent = func(c *ui.Context) { ui.Text(c, "owner-native-body") }
	}}}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 900)
	if err := tt.Click("Native plugin settings"); err != nil {
		t.Fatal(err)
	}
	if selected != 1 || !tt.HasText("owner-native-body") || activeSection(m, store.ViewState{}) != m.ActiveSection {
		t.Fatal("plugin section did not use the existing selected body and rail")
	}
	if err := tt.Click(i18n.T(i18n.KeySettingsSectionGeneral)); err != nil {
		t.Fatal(err)
	}
	if m.SectionContent != nil || !m.DesktopActive {
		t.Fatal("returning to a native page kept plugin body ownership")
	}
}
