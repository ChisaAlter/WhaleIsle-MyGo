package main

import (
	"os"
	"strings"
)

// buildVersion is set at link time by scripts/package.ps1 (-ldflags -X main.buildVersion=...).
var buildVersion = "dev"

// isLauncherPackage mirrors Electron status.launcherPackage (tray row gate).
func isLauncherPackage() bool {
	v := strings.TrimSpace(os.Getenv("WHALEISLE_LAUNCHER_PACKAGE"))
	if v == "1" || strings.EqualFold(v, "true") {
		return true
	}
	// Packaged NSIS/builds set a non-dev version via ldflags.
	return buildVersion != "" && buildVersion != "dev"
}
