package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/deeplink"
)

func TestDesktopTrayMenuDispatchAndSnapshot(t *testing.T) {
	var calls []string
	record := func(s string) func() { return func() { calls = append(calls, s) } }
	m := buildDesktopTrayMenu(desktopTrayActions{
		open: record("open"), launcher: record("launcher"), settings: record("settings"), market: record("market"), restart: record("restart"), quit: record("quit"),
		pet: func(enabled bool) {
			if !enabled {
				t.Fatal("checkbox did not pass explicit desired state")
			}
			calls = append(calls, "pet")
		},
		canRestart: true, canTogglePet: true,
	})
	for _, id := range []string{"open", "launcher", "settings", "market", "restart", "pet", "quit"} {
		item := m.ItemByID(id)
		if item == nil || item.Click == nil || item.Disabled {
			t.Fatalf("missing real action: %s", id)
		}
		if id == "pet" {
			if item.Type != mygo.MenuItemCheckbox || item.Checked {
				t.Fatal("wrong initial pet state")
			}
			item.Checked = true
		}
		item.Click(item, nil)
	}
	if !reflect.DeepEqual(calls, []string{"open", "launcher", "settings", "market", "restart", "pet", "quit"}) {
		t.Fatal(calls)
	}
	disabled := buildDesktopTrayMenu(desktopTrayActions{petEnabled: true})
	if !disabled.ItemByID("restart").Disabled || !disabled.ItemByID("pet").Disabled || !disabled.ItemByID("pet").Checked {
		t.Fatal("unavailable actions or current persisted pet state lost")
	}
}

func TestDesktopTrayNoticeHidesFirstAndPersistsOnce(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "tray-hide-acknowledged")
	var events []string
	hide := func() { events = append(events, "hide") }
	notify := func() error { events = append(events, "notice"); return nil }
	n := trayHideNotice{markerPath: marker}
	if err := n.close(hide, notify); err != nil {
		t.Fatal(err)
	}
	if err := n.close(hide, notify); err != nil {
		t.Fatal(err)
	}
	another := trayHideNotice{markerPath: marker}
	if err := another.close(hide, notify); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, []string{"hide", "notice", "hide", "hide"}) {
		t.Fatal(events)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal(err)
	}
}

func TestDesktopTrayNoticeFailureStillHidesOnlyOnce(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(parent, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	n := trayHideNotice{markerPath: filepath.Join(parent, "marker")}
	hidden, notices := 0, 0
	for i := 0; i < 2; i++ {
		_ = n.close(func() { hidden++ }, func() error { notices++; return errors.New("notifications unavailable") })
	}
	if hidden != 2 || notices != 1 {
		t.Fatalf("hidden=%d notices=%d", hidden, notices)
	}
	b, _ := os.ReadFile(parent)
	if string(b) != "keep" {
		t.Fatal("overwrote preexisting data")
	}
}

func TestDesktopTrayCloseDecisionRequiresRecoveryPath(t *testing.T) {
	for _, tc := range []struct {
		quitting, closeToTray, ready bool
		want                         desktopCloseAction
	}{
		{false, true, true, desktopCloseHide}, {false, true, false, desktopCloseKeep},
		{false, false, true, desktopCloseQuit}, {false, false, false, desktopCloseQuit},
		{true, true, true, desktopCloseQuit}, {true, true, false, desktopCloseQuit},
	} {
		if got := desktopCloseDecision(tc.quitting, tc.closeToTray, tc.ready); got != tc.want {
			t.Fatalf("%+v: %v", tc, got)
		}
	}
}

func TestDesktopTrayRoutesClearHiddenPanelConflicts(t *testing.T) {
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []deeplink.Action{deeplink.ActionLauncher, deeplink.ActionSettings, deeplink.ActionMarket} {
		s := &mainState{cfgStore: store, imOpen: true, remoteOpen: true, marketOpen: true, toolsMenuOpen: true, webviewOpen: true, browserPreviewOpen: true, assistantOpen: true}
		s.openDesktopTrayPage(action)
		if s.imOpen || s.remoteOpen || s.toolsMenuOpen || s.webviewOpen || s.browserPreviewOpen || s.assistantOpen {
			t.Fatalf("hidden route masks %s", action)
		}
		if (action == deeplink.ActionLauncher) != s.launcherOpen || (action == deeplink.ActionSettings) != s.settingsOpen || (action == deeplink.ActionMarket) != s.marketOpen {
			t.Fatalf("wrong target: %s", action)
		}
	}
	s := &mainState{quitting: true}
	s.openDesktopTrayPage(deeplink.ActionLauncher)
	if s.launcherOpen {
		t.Fatal("navigation reopened during final quit")
	}
}
