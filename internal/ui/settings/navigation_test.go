package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestSettingsFeatureSectionKeepsShellAndGeneralControls(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	closed := false
	locale := "en"
	m := Model{Desktop: &DesktopAccess{Fields: DesktopFields{Locale: &locale}}, DesktopActive: true, OnClose: func() { closed = true }}
	m.OnOpenMarket = func() {
		m.ActiveSection = SectionMarket
		m.SectionContent = func(c *ui.Context) { ui.Text(c, "Feature-owned market body") }
	}
	m.OnSelectDesktop = func() { m.ActiveSection = ""; m.SectionContent = nil; m.DesktopActive = true }
	// The market entry left the rail (task-48), so the feature section is reached
	// the way a deep link does it, not by clicking the navigation.
	m.OnOpenMarket()
	tt := ui.NewTester(func(c *ui.Context) {
		if !closed {
			ModalView(c, m)
		}
	}, 1000, 760)
	if !tt.HasText("Feature-owned market body") || !tt.HasText(i18n.T(i18n.KeySettingsTitle)) {
		t.Fatal("feature selection lost the shared settings shell")
	}
	tt.SetSize(600, 400)
	close, ok := tt.Find(i18n.T(i18n.KeySettingsBack))
	if !ok || close.X < 0 || close.Y < 0 || close.X+close.W > 600 || close.Y+close.H > 400 {
		t.Fatalf("close action outside smaller viewport: %+v, found=%v", close, ok)
	}
	if err := tt.Click(sectionLabel(SectionGeneral)); err != nil {
		t.Fatal(err)
	}
	// The reference General page labels the locale row "Language"; the old
	// "Interface language" wording was a MyGo-only drift.
	if !tt.HasText("Language") || tt.HasText("Feature-owned market body") {
		t.Fatal("General did not restore existing desktop controls")
	}
	tt.Key(0, ui.KeyEscape)
	if !closed {
		t.Fatal("Escape did not dismiss the shared settings dialog")
	}
}

func TestSettingsDedicatedSelectionWinsOverStaleSchemaSelection(t *testing.T) {
	snap := store.ViewState{SelectedNS: "pet"}
	if got := activeSection(Model{InterfaceActive: true, Interface: &InterfaceAccess{}}, snap); got != SectionInterface {
		t.Fatalf("stale pet namespace took selection from Interface: %s", got)
	}
	if got := activeSection(Model{InterfaceActive: true}, snap); got != SectionGeneral {
		t.Fatalf("unregistered Interface page bypassed navigation admission: %s", got)
	}
	for _, ns := range []string{"agent-loop", "web-search-deepseek", "subagent", "pwsh-local", "permission", "general"} {
		if got := activeSection(Model{}, store.ViewState{SelectedNS: ns}); got != SectionGeneral {
			t.Fatalf("%s became an extra primary section: %s", ns, got)
		}
	}
}

func TestSettingsUnsignedAccountStillHasLoginEntry(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	selected := false
	m := Model{Desktop: &DesktopAccess{}, DesktopActive: true, Account: &AccountAccess{}, OnSelectAccount: func() { selected = true }}
	for _, entry := range primaryNavigation(m, store.ViewState{}) {
		if entry.id == SectionAccount {
			t.Fatal("unsigned account unexpectedly occupies the primary rail")
		}
	}
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1000, 760)
	if err := tt.Click(i18n.T(i18n.KeySettingsAccountSignIn)); err != nil {
		t.Fatal(err)
	}
	if !selected {
		t.Fatal("General login entry lost its existing account callback")
	}
	m.AccountActive = true
	m.DesktopActive = false
	if got := activeSection(m, store.ViewState{}); got != SectionAccount {
		t.Fatalf("opened account flow lost selection: %s", got)
	}
	entries := primaryNavigation(m, store.ViewState{})
	if len(entries) == 0 || entries[0].id != SectionAccount {
		t.Fatal("active sign-in flow has no account navigation entry")
	}
}

// TestSettingsNavigationHidesMarketButDeepLinkResolves covers task-48: the market
// entry leaves the settings rail, while the constant, the namespace mapping and
// the rendering path stay so an existing deep link still lands on the page.
func TestSettingsNavigationRestoresMarketAndDeepLinkResolves(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	locale := "en"
	opened := 0
	m := Model{Desktop: &DesktopAccess{Fields: DesktopFields{Locale: &locale}}, DesktopActive: true, OnClose: func() {}, OnOpenMarket: func() { opened++ }}
	found := false
	for _, entry := range primaryNavigation(m, store.ViewState{}) {
		if entry.id == SectionMarket {
			found = true
		}
	}
	if !found {
		t.Fatal("the settings rail lost the market entry")
	}
	if !planSectionOrderHas(SectionMarket) {
		t.Fatal("the market section is missing from the plan order")
	}
	for _, alias := range []string{"market", "ui-settings-market"} {
		if got := SectionForNamespace(alias); got != SectionMarket {
			t.Fatalf("deep link %q resolved to %q, want %q", alias, got, SectionMarket)
		}
	}
	m.ActiveSection = SectionMarket
	m.SectionContent = func(c *ui.Context) { ui.Text(c, "Feature-owned market body") }
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1000, 760)
	if !tt.HasText("Feature-owned market body") {
		t.Fatal("a deep link to the market section no longer renders")
	}
	if err := tt.Click(sectionLabel(SectionMarket)); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("market rail entry calls=%d, want 1", opened)
	}
}

// planSectionOrderHas reports whether a section still appears in the rail order.
func planSectionOrderHas(id SectionID) bool {
	for _, candidate := range planSectionOrder {
		if candidate == id {
			return true
		}
	}
	return false
}
