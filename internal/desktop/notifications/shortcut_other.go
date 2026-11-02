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

const ToastActivatorCLSID = "{9CC711B1-3FB8-4C05-AB6C-CF32C5077B0F}"
const ToastServerFlag = "--whaleisle-toast-server"

func (s Shortcut) RegisterActivator() error          { return nil }
func SetProcessIdentity(appID string) error          { return nil }
func (s Shortcut) UnregisterOwnedIntegration() error { return nil }

// MigrateLegacy is Windows-only and leaves all files untouched elsewhere.
func (s Shortcut) MigrateLegacy(backupRoot string) (string, error) { return "", nil }

// ErrUnsupported reports whether shortcut registration exists on this platform.
func ErrUnsupported() error { return errNoShortcutShell }
