package main

import "testing"

func TestPrivateDesktopCannotHideWithoutTray(t *testing.T) {
	previous := portableDesktopRoot
	t.Cleanup(func() { portableDesktopRoot = previous })
	t.Setenv("WHALEISLE_ISOLATED", "")
	state := &mainState{closeToTray: true}
	portableDesktopRoot = ""
	if !state.shouldCloseToTray() {
		t.Fatal("installed desktop should retain its configured close policy")
	}
	portableDesktopRoot = t.TempDir()
	if state.shouldCloseToTray() {
		t.Fatal("portable desktop must close instead of hiding without a tray")
	}
	portableDesktopRoot = ""
	t.Setenv("WHALEISLE_ISOLATED", "1")
	if state.shouldCloseToTray() {
		t.Fatal("isolated QA desktop must not hide without a tray")
	}
}
