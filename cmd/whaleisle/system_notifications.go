package main

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo"
)

// Port of WhaleIsle@90465b67 src/main/system-notifications.js.
//
// The reference gate exists because raw Electron notifications register a Start
// Menu `Electron.lnk` with the binary's PE name and the installed AppID, and
// even `Notification.isSupported()` initializes that presenter. Its development
// boundary therefore precedes every Electron call.
//
// MyGo uses native WinRT toast XML and the registered Go-shell AppUserModelID.
// setupSystemNotifications registers its dedicated COM activator behind the
// desktop integration gate. Isolated/portable runs never use the product
// notification identity or mutate notification-center history.
//
// The Go WinRT presenter does not create Electron.lnk. Normal local builds may
// use their explicitly registered identity; isolated and portable profiles are
// gated separately before any support probe or setup.

// developmentApp mirrors Electron's process.defaultApp: an unpackaged build.
// scripts/package.ps1 stamps local packages as 0.0.0-dev, and an unstamped
// build keeps the `dev` default, so both are development builds.
func developmentApp() bool {
	version := strings.TrimSpace(buildVersion)
	return version == "" || version == "dev" || strings.HasSuffix(version, "-dev")
}

// systemNotificationsAllowed permits registered local builds as well as
// packages. Development mode needs no Electron presenter exception in MyGo.
// The isolated/portable boundary lives in systemNotificationsSupported.
func systemNotificationsAllowed(platform string, development bool) bool {
	return true
}

// systemNotificationsSupported is the single support check every notification
// site shares: the platform policy plus MyGo's own support probe (which already
// requires a packaged bundle on macOS).
func systemNotificationsSupported() bool {
	if isolatedDesktopRun() {
		return false
	}
	if err := setupSystemNotifications(); err != nil {
		return false
	}
	if !systemNotificationsAllowed(runtime.GOOS, developmentApp()) {
		return false
	}
	return mygo.NotificationsSupported()
}
