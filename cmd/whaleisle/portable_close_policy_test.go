package main

import (
	"testing"

	launcherctl "whaleisle/internal/app/launcher"
)

func TestPrivateDesktopCannotHideWithoutTray(t *testing.T) {
	previous := launcherctl.SetPortableDesktopRoot("")
	t.Cleanup(func() { launcherctl.SetPortableDesktopRoot(previous) })
	t.Setenv("WHALEISLE_ISOLATED", "")
	state := &mainState{closeToTray: true}
	if !state.shouldCloseToTray() {
		t.Fatal("installed desktop should retain its configured close policy")
	}
	launcherctl.SetPortableDesktopRoot(t.TempDir())
	if state.shouldCloseToTray() {
		t.Fatal("portable desktop must close instead of hiding without a tray")
	}
	launcherctl.SetPortableDesktopRoot("")
	t.Setenv("WHALEISLE_ISOLATED", "1")
	if state.shouldCloseToTray() {
		t.Fatal("isolated QA desktop must not hide without a tray")
	}
}
