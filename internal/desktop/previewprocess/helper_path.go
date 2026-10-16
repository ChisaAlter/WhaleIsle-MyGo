package previewprocess

import (
	"path/filepath"
	"runtime"
)

// HelperPath resolves only the companion shipped beside the trusted application.
// It never searches PATH or accepts a document-supplied location.
func HelperPath(executable string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(filepath.Dir(executable), "..", "Helpers", "WhaleIsle Preview.app", "Contents", "MacOS", "whaleisle-preview")
	}
	name := "whaleisle-preview"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(executable), name)
}
