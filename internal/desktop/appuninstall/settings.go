// Package appuninstall provides an explicit handoff to OS application management.
// It does not discover or execute uninstallers or report uninstall completion.
package appuninstall

import "errors"

var (
	ErrUnsupported = errors.New("installed apps settings is only available on Windows")
	ErrUnavailable = errors.New("installed apps settings opener is unavailable")
)

// OpenInstalledApps opens only the fixed Windows application-management page.
// No caller-supplied path, executable, argument, registry value or URI is accepted.
// A nil error means the OS accepted the open request, not that uninstall occurred.
func OpenInstalledApps(goos string, openExternal func(string) error) error {
	if goos != "windows" {
		return ErrUnsupported
	}
	if openExternal == nil {
		return ErrUnavailable
	}
	return openExternal("ms-settings:appsfeatures")
}
