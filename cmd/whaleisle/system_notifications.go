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
// MyGo's Windows notifications are notification-area balloons
// (third_party/mygo/internal/windows/tray.go: an existing tray, or a temporary
// one with the application icon), so nothing is registered and no shortcut is
// written. The equivalent visible prompt on this platform is the balloon/ toast
// itself; where MyGo cannot show it, the callers keep their in-app notice: the
// launcher row plus the frame flash for update confirmations, the Bot page
// status line for Bot events and the tray tooltip/menu for tray hiding.
//
// systemNotificationsAllowed is still the single policy used off Windows, so a
// caller never reasons about the platform boundary itself.

// developmentApp mirrors Electron's process.defaultApp: an unpackaged build.
// scripts/package.ps1 stamps local packages as 0.0.0-dev, and an unstamped
// build keeps the `dev` default, so both are development builds.
func developmentApp() bool {
	version := strings.TrimSpace(buildVersion)
	return version == "" || version == "dev" || strings.HasSuffix(version, "-dev")
}

// systemNotificationsAllowed reports whether this platform may show system
// notifications at all. Packaged applications and every platform but Windows
// keep the policy; see the package comment for why Windows is exempt here.
func systemNotificationsAllowed(platform string, development bool) bool {
	return platform != "windows" || !development
}

// systemNotificationsSupported is the single support check every notification
// site shares: the platform policy plus MyGo's own support probe (which already
// requires a packaged bundle on macOS).
func systemNotificationsSupported() bool {
	if runtime.GOOS != "windows" && !systemNotificationsAllowed(runtime.GOOS, developmentApp()) {
		return false
	}
	return mygo.NotificationsSupported()
}
