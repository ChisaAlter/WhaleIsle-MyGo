package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
)

func immediateDesktopState(t *testing.T) (*mainState, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Update(func(c *config.Config) {
		c.CloseToTray, c.AutoStartDesktop, c.OpenAtLogin = true, false, false
		c.Locale, c.Workspace, c.Port = "en", "committed-workspace", 4999
	}); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store, closeToTray: true}
	s.seedDesktopDraft()
	// These unrelated unsaved choices would make the full form save invalid.
	s.desktopDraft.port, s.desktopDraft.harnessRestartMaxAttempts, s.desktopDraft.harnessRestartBaseDelayMs = "not-a-port", "invalid", "invalid"
	s.desktopDraft.workspace, s.desktopDraft.openAtLogin, s.desktopDraft.locale = "unsaved-workspace", true, "zh"
	s.desktopSaveErr, s.desktopLastAction = "unrelated form error", "unrelated form action"
	return s, dir
}

func TestDesktopImmediateBoolUICommitsSingleFieldAndReopens(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	s, dir := immediateDesktopState(t)
	before, draft := s.cfgStore.Get(), s.desktopDraft
	formSaves := 0
	tt := ui.NewTester(func(c *ui.Context) {
		settings.View(c, settings.Model{DesktopActive: true, ThemeID: "deepseek.light", Desktop: &settings.DesktopAccess{
			Fields: settings.DesktopFields{CloseToTray: &s.desktopDraft.closeToTray}, AutoStartDesktop: &s.desktopDraft.autoStartDesktop,
			OnSetCloseToTray: s.setCloseToTrayPreference, OnSetAutoStartDesktop: s.setAutoStartDesktopPreference,
			OnSave: func() { formSaves++ },
		}})
	}, 1100, 1600)
	choose := func(label, option string) {
		t.Helper()
		current := "Open launcher first"
		if s.desktopDraft.autoStartDesktop {
			current = "Start desktop directly"
		}
		if label == "When closing the window" {
			current = "Quit"
			if s.desktopDraft.closeToTray {
				current = "Minimize to tray"
			}
		}
		if err := tt.Click(current); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(option); err != nil {
			t.Fatal(err)
		}
	}
	assertStored := func(closeToTray, autoStart bool) {
		t.Helper()
		reopened, err := config.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		want := before
		want.CloseToTray, want.AutoStartDesktop = closeToTray, autoStart
		if got := reopened.Get(); !reflect.DeepEqual(got, want) {
			t.Fatalf("single-field save changed other committed settings: got=%+v want=%+v", got, want)
		}
	}
	choose("When closing the window", "Quit")
	assertStored(false, false)
	if s.closeToTray || s.desktopDraft.closeToTray {
		t.Fatal("accepted close choice did not update current close behavior and draft")
	}
	choose(i18n.T(i18n.KeySettingsInterfaceAutoStart), "Start desktop directly")
	assertStored(false, true)
	if !s.desktopDraft.autoStartDesktop {
		t.Fatal("accepted startup choice did not publish")
	}
	draft.closeToTray, draft.autoStartDesktop = false, true
	if s.desktopDraft != draft || formSaves != 0 || s.desktopSaveErr != "unrelated form error" || s.desktopLastAction != "unrelated form action" {
		t.Fatal("immediate save touched an unrelated draft or submitted its Save")
	}
	// No app/Host/native backend has been initialized: reaching OS login
	// registration or process startup here would fail the test.
}

func TestDesktopImmediateBoolWriteFailureLeavesActiveAndDraftUntouched(t *testing.T) {
	for _, closeField := range []bool{true, false} {
		name := "startup"
		if closeField {
			name = "close"
		}
		t.Run(name, func(t *testing.T) {
			// Open a not-yet-created config, then occupy its parent with an owned
			// regular file. Saving must fail without requiring permission changes
			// or a production persistence test hook.
			blocker := filepath.Join(t.TempDir(), "parent-file")
			store, err := config.Open(blocker)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(blocker, []byte("owned blocker"), 0600); err != nil {
				t.Fatal(err)
			}
			before := store.Get()
			s := &mainState{cfgStore: store, closeToTray: before.CloseToTray}
			s.seedDesktopDraft()
			draft := s.desktopDraft
			set, want := s.setAutoStartDesktopPreference, before.AutoStartDesktop
			if closeField {
				set, want = s.setCloseToTrayPreference, before.CloseToTray
			}
			accepted, err := set(!want)
			if err == nil || accepted != want {
				t.Fatalf("accepted=%v error=%v", accepted, err)
			}
			if s.closeToTray != before.CloseToTray || s.desktopDraft != draft || !reflect.DeepEqual(s.cfgStore.Get(), before) {
				t.Fatal("failed write changed active state or drafts")
			}
			if raw, err := os.ReadFile(blocker); err != nil || string(raw) != "owned blocker" {
				t.Fatal("failed save changed parent file", err)
			}
		})
	}
}

func TestDesktopImmediateBoolRejectsStaleConfigWithoutOverwriting(t *testing.T) {
	s, dir := immediateDesktopState(t)
	external, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = external.Update(func(c *config.Config) { c.Workspace = "external-workspace" }); err != nil {
		t.Fatal(err)
	}
	rawBefore, err := os.ReadFile(external.Path())
	if err != nil {
		t.Fatal(err)
	}
	draft := s.desktopDraft
	accepted, err := s.setCloseToTrayPreference(false)
	if !errors.Is(err, config.ErrConflict) || !accepted || !s.closeToTray || s.desktopDraft != draft {
		t.Fatalf("stale accepted=%v error=%v", accepted, err)
	}
	rawAfter, err := os.ReadFile(external.Path())
	if err != nil || !bytes.Equal(rawBefore, rawAfter) {
		t.Fatal("conflict replaced external config", err)
	}
}
