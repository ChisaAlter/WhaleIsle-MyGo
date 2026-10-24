// Package notifications owns the desktop notification identity.
//
// Native Electron registers a Windows AppUserModelID ("ai.deepseek.harness.gui",
// reference/electron/src/main/index.js setAppUserModelId, and
// legacy-notification-shortcut.js migrates the raw-Electron Start-menu link that
// carries it) so toasts show the product name instead of the binary name. This
// package performs the same registration for the MyGo shell: a Start-menu
// shortcut whose target is the running executable and whose
// System.AppUserModel.ID property is the shared app id.
package notifications

import (
	"os"
	"path/filepath"
)

// AppUserModelID follows the desktop build contract asserted by
// reference/electron/src/main/package-contract.test.js.
const AppUserModelID = "ai.deepseek.harness.gui"

// ShortcutName is the Start-menu link the shell owns.
const ShortcutName = "Whale Isle.lnk"

// StartMenuProgramsDir returns %APPDATA%\Microsoft\Windows\Start Menu\Programs.
// The empty string means the location is unknown (no APPDATA, non-Windows).
func StartMenuProgramsDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs")
}

// Shortcut is the Start-menu link that registers AppUserModelID for the shell.
type Shortcut struct {
	ProgramsDir string
	// Target is the executable the link launches (normally os.Executable()).
	Target string
}

// Path is the link location inside the Programs directory.
func (s Shortcut) Path() string {
	if s.ProgramsDir == "" {
		return ""
	}
	return filepath.Join(s.ProgramsDir, ShortcutName)
}

// Ensure writes or refreshes the link. It is a no-op when the location or the
// target is unknown, and delegates the platform work to createShortcut.
func (s Shortcut) Ensure() error { return s.ensureWith(createShortcut) }

// ensureWith is the shell seam: tests substitute a recorder for createShortcut.
func (s Shortcut) ensureWith(create func(lnk, target, appID string) error) error {
	lnk, target := s.Path(), s.Target
	if lnk == "" || target == "" {
		return nil
	}
	return create(lnk, target, AppUserModelID)
}
