//go:build !windows

package paths

import (
	"os"
	"path/filepath"
)

// osDocumentsDir is the non-Windows Documents folder: ~/Documents, the same
// place the reference shell resolves it from.
func osDocumentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}
