package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
)

func TestDeferredApplicationMenuRejectsStaleRevisionAndNewModal(t *testing.T) {
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg, applicationMenu: &desktopApplicationMenu{}}
	prefs := s.pageShortcutPreferences()
	revision := prefs.Get().Revision
	modal := false
	tt := ui.NewTester(func(c *ui.Context) {
		defer s.processApplicationMenu(c)
		ui.Text(c, "fixture page")
		if modal {
			open := true
			ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
				panel.Width(200).Height(100).Children(func() { ui.Text(c, "new modal") })
			})
		}
	}, 480, 320)
	s.applicationMenu.pending = []desktopMenuGesture{{"settings.about", revision}}
	if s.settingsOpen {
		t.Fatal("queued menu action ran before a frame")
	}
	// Change accepted preferences after the native click, before its next frame.
	result := prefs.Apply("native", revision, []shortcutprefs.Edit{{Type: "set", ID: shortcuts.CanonicalID("window.zoomIn"), Binding: &shortcutprefs.Binding{Code: "Digit9", Modifiers: []string{"primary", "alt"}}}})
	if result.Status != "saved" {
		t.Fatal(result.Status)
	}
	tt.Frame()
	if s.settingsOpen || len(s.applicationMenu.pending) != 0 {
		t.Fatal("stale menu revision executed or remained queued")
	}
	revision = prefs.Get().Revision
	s.applicationMenu.pending = []desktopMenuGesture{{"settings.about", revision}}
	modal = true
	tt.Frame()
	if s.settingsOpen || len(s.applicationMenu.pending) != 0 {
		t.Fatal("menu ran through a newly built modal")
	}
	modal = false
	tt.Frame() // Retire the modal before testing an allowed gesture.
	s.applicationMenu.pending = []desktopMenuGesture{{"settings.about", revision}}
	tt.Frame()
	if !s.settingsOpen || !s.settingsAboutActive {
		t.Fatal("current unblocked menu gesture failed to use the live command")
	}
}

func TestDeferredApplicationMenuRecordingAndResetOwnInput(t *testing.T) {
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg, applicationMenu: &desktopApplicationMenu{}}
	e := s.pageShortcutEditor()
	revision := s.pageShortcutPreferences().Get().Revision
	tt := ui.NewTester(func(c *ui.Context) { ui.Text(c, "fixture"); s.processApplicationMenu(c) }, 400, 220)
	e.BeginRecording("window.refresh")
	s.applicationMenu.pending = []desktopMenuGesture{{"settings.about", revision}}
	tt.Frame()
	if s.settingsOpen {
		t.Fatal("menu bypassed recording")
	}
	e.RequestReset()
	s.applicationMenu.pending = []desktopMenuGesture{{"settings.about", revision}}
	tt.Frame()
	if s.settingsOpen {
		t.Fatal("menu bypassed reset confirmation")
	}
	e.CancelReset()
	tt.Frame()
	if s.settingsOpen {
		t.Fatal("rejected gesture replayed after reset cancellation")
	}
}

func TestCapturedProductCloseDropsExpiredLeaseAndBlockedFrame(t *testing.T) {
	for _, reason := range []string{"stale-lease", "new-modal", "recording", "reset", "quitting"} {
		t.Run(reason, func(t *testing.T) {
			s := &mainState{}
			checks := 0
			// This opaque pointer must never reach native methods in a rejected branch.
			owner := new(mygo.Window)
			s.productShortcutCloses = []productShortcutClose{{owner: owner, current: func() bool { checks++; return false }}}
			if reason == "recording" || reason == "reset" {
				s.shortcutEditor = shortcuts.NewEditor(nil, nil, nil)
				if reason == "recording" {
					s.shortcutEditor.BeginRecording("window.refresh")
				} else {
					s.shortcutEditor.RequestReset()
				}
			}
			s.quitting = reason == "quitting"
			_ = ui.NewTester(func(c *ui.Context) {
				defer s.processProductShortcutClose(c)
				ui.Text(c, "fixture")
				if reason == "new-modal" {
					open := true
					ui.DialogBase(c, &open, func(backdrop, panel ui.Element) { panel.Width(200).Height(100) })
				}
			}, 400, 220)
			if len(s.productShortcutCloses) != 0 {
				t.Fatal("rejected request retained for replay")
			}
			want := 0
			if reason == "stale-lease" {
				want = 1
			}
			if checks != want {
				t.Fatal("wrong close decision boundary", checks, want)
			}
		})
	}
	s := &mainState{}
	s.queueProductShortcutClose(nil, func() bool { return true })
	s.queueProductShortcutClose(new(mygo.Window), nil)
	if len(s.productShortcutCloses) != 0 {
		t.Fatal("uncaptured product close accepted")
	}
}
