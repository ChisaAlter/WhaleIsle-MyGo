package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// detailHolds opens the first 详情 disclosure, checks the store's own words
// behind it, and collapses it again so the layout the caller measures returns to
// what it was. It reports false when the page stands those words in the message
// position, or offers no disclosure to open.
func detailHolds(t *testing.T, tt *ui.Tester, raw string) bool {
	t.Helper()
	if tt.HasText(raw) {
		return false
	}
	if err := tt.Click(i18n.Pick("详情", "Details")); err != nil {
		return false
	}
	tt.Frame()
	held := tt.HasText(raw)
	if err := tt.Click(i18n.Pick("收起", "Hide")); err == nil {
		tt.Frame()
	}
	return held
}

func TestSettingsAlertKeepsStoreWordsBehindDetail(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	raw := "settings/describe: dial tcp 127.0.0.1:7788: connectex: No connection could be made because the target machine actively refused it."
	a := &AboutAccess{AppName: "WhaleIsle", Version: "0.0.0-fixture", Err: raw}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			aboutPane(c, Model{About: a}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 620, 900)
	if !tt.HasText("Update information could not be read. Check the connection and retry.") {
		t.Fatalf("the alert does not say what happened and what to do: %v", tt.Texts())
	}
	if !detailHolds(t, tt, raw) {
		t.Fatalf("the store's words are not reachable behind 详情: %v", tt.Texts())
	}
}

func TestSettingsFailureLineKeepsStoreWordsBehindDetail(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	raw := "read C:/Users/fixture/AppData/Roaming/WhaleIsle/companion.json: The process cannot access the file."
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			settingsFailureLine(c, "spot/line", "The desktop character settings could not be read. Try again.", raw)
		})
	}, 620, 200)
	if !tt.HasText("The desktop character settings could not be read. Try again.") {
		t.Fatalf("the line does not say what happened and what to do: %v", tt.Texts())
	}
	if !detailHolds(t, tt, raw) {
		t.Fatalf("the store's words are not reachable behind 详情: %v", tt.Texts())
	}
}

func TestPreferenceFieldAlertKeepsStoreWordsBehindDetail(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	raw := "settings/mutate ui-conversation.viewTabs: error code 40002: invalid arguments"
	h := bodyFixtureHost()
	h.Snapshot.PreferenceErrors = map[string]string{"ui-conversation.viewTabs": raw}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			interfacePane(c, Model{Interface: &InterfaceAccess{Host: h}}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 1000, 900)
	if !tt.HasText(preferenceFailureCopy("Conversation and trajectory tabs")) {
		t.Fatalf("the field alert does not name the setting that failed: %v", tt.Texts())
	}
	if !detailHolds(t, tt, raw) {
		t.Fatalf("the store's words are not reachable behind 详情: %v", tt.Texts())
	}
}
