package main

import (
	"image/color"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
)

func TestMainNavigationRendersCurrentRouteAndCommittedShortcut(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	profile := "desktop:" + shortcutPlatform()
	var savedShortcuts []byte
	prefs := shortcutprefs.New(profile, func() ([]byte, error) { return append([]byte(nil), savedShortcuts...), nil }, func(data []byte) error {
		savedShortcuts = append([]byte(nil), data...)
		return nil
	})
	prefs.SetDefinitions("native", shortcuts.SharedDefinitions(profile))
	store := session.New()
	cfgStore, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{sessions: store, shortcutPreferences: prefs, cfgStore: cfgStore}
	sizes := frame.DefaultLayout()
	query, rename, draft := "", "", ""
	tt := ui.NewTester(func(c *ui.Context) {
		s.withMainNavigation(c, true, frame.Model{
			Store: store, Sizes: &sizes, ThemeID: "deepseek.light", SessionSearch: &query,
			RenameSessionID: &rename, RenameDraft: &draft,
		}, func() { ui.Text(c, "Resident conversation fixture") })
	}, 1200, 800)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	labels := []string{i18n.T(i18n.KeySidebarMarket), i18n.T(i18n.KeySidebarSchedule), i18n.T(i18n.KeySidebarAssistant)}
	idle, hovered := map[string]color.RGBA{}, map[string]color.RGBA{}
	for _, label := range labels {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatal("missing navigation", label)
		}
		tt.Move(1199, 700)
		idle[label] = tt.Image().RGBAAt(258, int(r.Y+11))
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		hovered[label] = tt.Image().RGBAAt(258, int(r.Y+11))
		if hovered[label] == idle[label] {
			t.Fatal("navigation hover absent", label)
		}
	}
	check := func(selected int) {
		t.Helper()
		tt.Move(1199, 700)
		tt.Frame()
		for index, label := range labels {
			r, ok := tt.Find(label)
			if !ok {
				t.Fatal("missing navigation", label)
			}
			want := idle[label]
			if index == selected {
				want = hovered[label]
			}
			got := tt.Image().RGBAAt(258, int(r.Y+11))
			if got != want {
				t.Fatalf("selected%d row%d paint=%v want=%v", selected, index, got, want)
			}
		}
	}
	for index, label := range labels {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		// Whale's source sidebar action opens its canonical conversation, not
		// the separate native management panel. Without a ready Host it stays
		// on the conversation and reports that readiness is required.
		if s.marketOpen != (index == 0) || s.scheduleOpen != (index == 1) || s.assistantOpen {
			t.Fatal("real route callback did not select its panel")
		}
		if index == 2 {
			check(-1)
		} else {
			check(index)
		}
	}
	if runtime.GOOS != "darwin" {
		if err := tt.Click(product.Name); err != nil {
			t.Fatal(err)
		}
		if s.marketOpen || s.scheduleOpen || s.assistantOpen {
			t.Fatal("brand action did not reveal the conversation")
		}
		check(-1)
	}
	label := i18n.T(i18n.KeySidebarNewSessionFallback)
	r, ok := tt.Find(label)
	if !ok {
		t.Fatal("new session missing")
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	display := func() string {
		value := shortcuts.Resolve(s.pageShortcutBindings())["session.new"].String()
		if runtime.GOOS == "windows" {
			return strings.Replace(value, "Mod", "Ctrl", 1)
		}
		if runtime.GOOS == "darwin" {
			return strings.Replace(value, "Mod", "Cmd", 1)
		}
		return value
	}
	initialHint := display()
	if !tt.HasText(initialHint) {
		t.Fatalf("initial shared default not displayed: %v", tt.Texts())
	}
	binding := &shortcutprefs.Binding{Code: "F8", Modifiers: []string{"primary", "alt"}}
	result := prefs.Apply("native", prefs.Accepted().Revision, []shortcutprefs.Edit{{Type: "set", ID: "session.new", Binding: binding}})
	if result.Status != "saved" {
		if result.Snapshot.Error != nil {
			t.Fatalf("fixture preference edit status=%s error=%s issue=%s", result.Status, *result.Snapshot.Error, result.Issue)
		}
		t.Fatalf("fixture preference edit=%+v", result)
	}
	tt.Frame()
	customHint := display()
	if !tt.HasText(customHint) || tt.HasText(initialHint) {
		t.Fatal("committed binding edit did not reach visible hint")
	}
	result = prefs.Apply("native", prefs.Accepted().Revision, []shortcutprefs.Edit{{Type: "set", ID: "session.new", Binding: nil}})
	if result.Status != "saved" {
		t.Fatalf("fixture unbind=%+v", result)
	}
	tt.Frame()
	if tt.HasText(customHint) {
		t.Fatal("committed unbind retained stale hint")
	}
}
