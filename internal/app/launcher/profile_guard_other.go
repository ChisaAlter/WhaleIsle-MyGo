//go:build !windows

package launcherctl

// ClaimDesktopProfile returns a no-op releaser on non-Windows hosts where the
// OS shell retains native single-instance locking.
func ClaimDesktopProfile() (func(), error) { return func() {}, nil }
