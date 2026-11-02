package settings

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestSettingsInstalledRailAndOptionsGeometry(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	m := Model{ActiveSection: SectionGeneral, SectionContent: func(c *ui.Context) { ui.Box(c).Label("Options origin").Size(8, 8) }, Account: &AccountAccess{}, Appearance: &AppearanceAccess{}, Interface: &InterfaceAccess{}, OnClose: func() {}}
	tt := ui.NewTester(func(c *ui.Context) { ModalView(c, m) }, 1440, 864)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))
	for _, expected := range []struct {
		label               string
		x, y, width, height float32
	}{{"通用设置", 332, 168, 164, 36}, {"外观", 332, 206, 164, 36}, {"界面设置", 332, 244, 164, 36}, {"Options origin", 532, 86, 8, 8}} {
		got, found := tt.Find(expected.label)
		if !found || math.Abs(float64(got.X-expected.x)) > .1 || math.Abs(float64(got.Y-expected.y)) > .1 || math.Abs(float64(got.W-expected.width)) > .1 || math.Abs(float64(got.H-expected.height)) > .1 {
			t.Fatalf("%s actual=%+v found=%v source=(%g,%g,%g,%g)", expected.label, got, found, expected.x, expected.y, expected.width, expected.height)
		}
	}
	// The source's narrow modal keeps the rail while the options shrink. Close
	// and its body remain in the client, with no stale wide-frame coordinates.
	tt.SetSize(600, 440)
	for _, label := range []string{i18n.T(i18n.KeySettingsBack), "Options origin"} {
		got, found := tt.Find(label)
		if !found || got.X < 0 || got.X+got.W > 600 || got.Y < 0 || got.Y+got.H > 440 {
			t.Fatalf("narrow %s outside client: %+v found=%v", label, got, found)
		}
	}
}

func TestSettingsAccountContributionMatchesStoredCredentials(t *testing.T) {
	for _, tc := range []struct {
		status          string
		active, visible bool
	}{{"", false, false}, {"signed-out", false, false}, {"credential-stored", false, true}, {"", true, true}} {
		m := Model{Desktop: &DesktopAccess{}, Account: &AccountAccess{Status: tc.status}, AccountActive: tc.active}
		found := false
		for _, entry := range primaryNavigation(m, store.ViewState{}) {
			if entry.id == SectionAccount {
				found = true
			}
		}
		if found != tc.visible {
			t.Fatalf("status=%q active=%v rail=%v want%v", tc.status, tc.active, found, tc.visible)
		}
		if tc.active && activeSection(m, store.ViewState{}) != SectionAccount {
			t.Fatal("explicit unsigned account route no longer admitted")
		}
	}
}

func TestSettingsInterfaceKeepsOnlyTheInstalledRows(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	host := bodyFixtureHost()
	host.Snapshot.Described = true
	bots := false
	access := &InterfaceAccess{Host: host, Project: &ProjectPreferenceAccess{}, DshbotEnabled: &bots, OnSetDshbot: func(bool) {}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { interfacePane(c, Model{Interface: access}, c.Theme().Text, c.Theme().TextMuted) })
	}, 800, 1800)
	// Project, Bots and the remote workspace live on the built-in features page;
	// next-prompt on the conversation page; the glow on the appearance page.
	for _, moved := range []string{"Project", "Bots", "Next prompt suggestions", "Thinking glow when sending", "Remote workspaces"} {
		if tt.HasText(moved) {
			t.Fatalf("Interface still carries %q: %v", moved, tt.Texts())
		}
	}
	git, g := tt.Find("Titlebar Git actions")
	stats, s := tt.Find("Session statistics")
	tabs, v := tt.Find("Conversation and trajectory tabs")
	if !g || !s || !v || !(git.Y < stats.Y && stats.Y < tabs.Y) {
		t.Fatalf("installed row order Git/stats/tabs: Git%+v stats%+v tabs%+v texts=%v", git, stats, tabs, tt.Texts())
	}
}
