package settings

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestDesktopImmediatePreferencesSaveIndependentlyAndRollback(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	closeToTray, autoStart := true, false
	closeCalls, startCalls, formSaves := 0, 0, 0
	var failure error
	mismatch := false
	desktop := &DesktopAccess{
		Fields: DesktopFields{CloseToTray: &closeToTray}, AutoStartDesktop: &autoStart,
		OnSetCloseToTray: func(next bool) (bool, error) {
			closeCalls++
			if failure != nil || mismatch {
				return closeToTray, failure
			}
			return next, nil
		},
		OnSetAutoStartDesktop: func(next bool) (bool, error) {
			startCalls++
			if failure != nil || mismatch {
				return autoStart, failure
			}
			return next, nil
		},
		OnSave: func() { formSaves++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { desktopPane(c, Model{Desktop: desktop}, c.Theme().Text, c.Theme().TextMuted) }, 900, 1300)
	choose := func(label, option string) {
		t.Helper()
		current := "Open launcher first"
		if autoStart {
			current = "Start desktop directly"
		}
		if label == "When closing the window" {
			current = "Quit"
			if closeToTray {
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
	choose("When closing the window", "Quit")
	if closeToTray || autoStart || closeCalls != 1 || startCalls != 0 || formSaves != 0 {
		t.Fatal("close selection did not save only its own field")
	}
	choose(i18n.T(i18n.KeySettingsInterfaceAutoStart), "Start desktop directly")
	if closeToTray || !autoStart || closeCalls != 1 || startCalls != 1 || formSaves != 0 {
		t.Fatal("startup selection submitted the form or changed close behavior")
	}
	failure = errors.New("synthetic config write failure")
	choose("When closing the window", "Minimize to tray")
	if closeToTray || closeCalls != 2 || !tt.HasText("Quit") || !tt.HasText(failure.Error()) {
		t.Fatal("failed close save did not restore accepted choice and show error")
	}
	choose(i18n.T(i18n.KeySettingsInterfaceAutoStart), "Open launcher first")
	if !autoStart || startCalls != 2 || !tt.HasText("Start desktop directly") {
		t.Fatal("failed startup save did not restore accepted choice")
	}
	failure = nil
	mismatch = true
	choose("When closing the window", "Minimize to tray")
	if closeToTray || !tt.HasText("The setting was not saved; retry.") {
		t.Fatal("unaccepted response shown as saved")
	}
	mismatch = false
	choose("When closing the window", "Minimize to tray")
	if !closeToTray || !autoStart || closeCalls != 4 || startCalls != 2 || formSaves != 0 {
		t.Fatal("retry did not commit only close behavior")
	}
}

func TestDesktopImmediatePreferencesWithoutFormSave(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	closeToTray, autoStart := true, false
	calls := 0
	tt := ui.NewTester(func(c *ui.Context) {
		desktopPane(c, Model{Desktop: &DesktopAccess{Fields: DesktopFields{CloseToTray: &closeToTray}, AutoStartDesktop: &autoStart,
			OnSetCloseToTray:      func(next bool) (bool, error) { calls++; return next, nil },
			OnSetAutoStartDesktop: func(next bool) (bool, error) { calls++; return next, nil },
		}}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1300)
	for _, choice := range []struct{ label, option string }{{"Minimize to tray", "Quit"}, {"Open launcher first", "Start desktop directly"}} {
		if err := tt.Click(choice.label); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(choice.option); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 || closeToTray || !autoStart {
		t.Fatal("immediate controls incorrectly depend on the form Save callback")
	}
}
