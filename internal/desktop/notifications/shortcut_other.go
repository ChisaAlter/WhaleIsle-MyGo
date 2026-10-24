//go:build !windows

package notifications

import "errors"

var errNoShortcutShell = errors.New("notifications: Start-menu shortcut is Windows-only")

// createShortcut is a no-op off Windows: the shell has no AppUserModelID and
// the tray/menu identity path does not exist there.
func createShortcut(lnk, target, appID string) error {
	_ = lnk
	_ = target
	_ = appID
	return nil
}

// ErrUnsupported reports whether shortcut registration exists on this platform.
func ErrUnsupported() error { return errNoShortcutShell }
