package main

import (
	"os"
	"path/filepath"
	"strings"
)

// Only the matching installation may accept an external launcher's stop
// request. The existing quitProtected path still owns all task confirmations.
func managedLauncherStop(args []string) bool {
	if len(args) != 2 || args[0] != "--managed-launcher-stop" {
		return false
	}
	current, err := os.Executable()
	if err != nil {
		return false
	}
	return filepath.IsAbs(args[1]) && strings.EqualFold(filepath.Clean(args[1]), filepath.Clean(current))
}
