package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
)

func TestSetLocalePreferenceRoundTripAndPersistence(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })

	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(c *config.Config) { c.Locale = "zh" }); err != nil {
		t.Fatal(err)
	}
	i18n.SetLocale("zh")
	s := &mainState{cfgStore: store}
	s.seedDesktopDraft()
	t.Cleanup(func() { _ = s.flushPendingConfigWrites() })

	steps := []struct {
		input      string
		wantLang   string
		wantLocale string
	}{
		{"en", "en", i18n.LocaleEN},
		{"zh", "zh", i18n.LocaleZH},
		{"en", "en", i18n.LocaleEN},
		{"zh-CN", "zh", i18n.LocaleZH},
		{"en-US", "en", i18n.LocaleEN},
		{"zh", "zh", i18n.LocaleZH},
		// Unsupported values must keep the current effective locale instead of corrupting it.
		{"fr", "zh", i18n.LocaleZH},
	}
	for _, step := range steps {
		accepted, err := s.setLocalePreference(step.input)
		if err != nil {
			t.Fatalf("setLocalePreference(%q) error: %v", step.input, err)
		}
		if accepted != step.wantLang {
			t.Fatalf("setLocalePreference(%q) = %q, want %q", step.input, accepted, step.wantLang)
		}
		if got := i18n.Locale(); got != step.wantLocale {
			t.Fatalf("after %q: i18n.Locale() = %q, want %q", step.input, got, step.wantLocale)
		}
		if got := s.desktopDraft.locale; got != step.wantLang {
			t.Fatalf("after %q: desktopDraft.locale = %q, want %q", step.input, got, step.wantLang)
		}
		if got := s.effectiveConfigFor(store).Locale; got != step.wantLang {
			t.Fatalf("after %q: effectiveConfigFor.Locale = %q, want %q", step.input, got, step.wantLang)
		}
		if err := s.flushPendingConfigWrites(); err != nil {
			t.Fatalf("flush after %q: %v", step.input, err)
		}
		reopened, err := config.Open(dir)
		if err != nil {
			t.Fatalf("reopen config after %q: %v", step.input, err)
		}
		if got := reopened.Get().Locale; got != step.wantLang {
			t.Fatalf("persisted config.json Locale after %q = %q, want %q", step.input, got, step.wantLang)
		}
	}
}

func TestSettingsLocaleSelectorRoundTripWithRealCallback(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })

	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(c *config.Config) { c.Locale = "zh" }); err != nil {
		t.Fatal(err)
	}
	i18n.SetLocale("zh")
	s := &mainState{cfgStore: store}
	s.seedDesktopDraft()
	t.Cleanup(func() { _ = s.flushPendingConfigWrites() })

	tt := ui.NewTester(func(c *ui.Context) {
		settings.View(c, settings.Model{
			DesktopActive: true,
			ThemeID:       "deepseek.light",
			Desktop: &settings.DesktopAccess{
				Fields:      settings.DesktopFields{Locale: &s.desktopDraft.locale},
				OnSetLocale: s.setLocalePreference,
			},
		})
	}, 1100, 1400)

	// Switch zh -> en via real settings UI selector.
	if err := tt.Click("中文"); err != nil {
		t.Fatalf("open locale select (zh): %v", err)
	}
	if err := tt.Click("English"); err != nil {
		t.Fatalf("pick English: %v", err)
	}
	if got := i18n.Locale(); got != i18n.LocaleEN {
		t.Fatalf("after zh->en UI click: i18n.Locale() = %q, want %q", got, i18n.LocaleEN)
	}
	if got := s.desktopDraft.locale; got != "en" {
		t.Fatalf("after zh->en UI click: draft = %q, want en", got)
	}

	// Switch en -> zh via real settings UI selector (regression guard for zh vs zh-CN bug).
	if err := tt.Click("English"); err != nil {
		t.Fatalf("open locale select (en): %v", err)
	}
	if err := tt.Click("中文"); err != nil {
		t.Fatalf("pick 中文: %v", err)
	}
	if got := i18n.Locale(); got != i18n.LocaleZH {
		t.Fatalf("after en->zh UI click: i18n.Locale() = %q, want %q", got, i18n.LocaleZH)
	}
	if got := s.desktopDraft.locale; got != "zh" {
		t.Fatalf("after en->zh UI click: draft = %q, want zh", got)
	}

	// Switch zh -> en again to verify repeated round-trips never lock up.
	if err := tt.Click("中文"); err != nil {
		t.Fatalf("open locale select (zh 2nd): %v", err)
	}
	if err := tt.Click("English"); err != nil {
		t.Fatalf("pick English (2nd): %v", err)
	}
	if got := i18n.Locale(); got != i18n.LocaleEN {
		t.Fatalf("after 2nd zh->en UI click: i18n.Locale() = %q, want %q", got, i18n.LocaleEN)
	}

	if err := s.flushPendingConfigWrites(); err != nil {
		t.Fatalf("flush config: %v", err)
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Get().Locale; got != "en" {
		t.Fatalf("persisted Locale = %q, want en", got)
	}
}
