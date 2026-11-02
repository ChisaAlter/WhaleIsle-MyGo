//go:build windows

// whaleisle-integration is a small windowsgui installation helper. It does not
// launch the desktop or Host and is executed from the installer's own embedded
// copy, never an installed old executable.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/notifications"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "--remove-owned-notifications" {
		os.Exit(2)
	}
	target := os.Args[2]
	if !filepath.IsAbs(target) || !strings.EqualFold(filepath.Base(target), "whaleisle.exe") {
		os.Exit(2)
	}
	s := notifications.Shortcut{ProgramsDir: notifications.StartMenuProgramsDir(), Target: filepath.Clean(target)}
	if err := s.UnregisterOwnedIntegration(); err != nil {
		os.Exit(1)
	}
}
