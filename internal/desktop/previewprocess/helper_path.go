package previewprocess

import (
	"path/filepath"
	"runtime"
)

// HelperPath resolves only the companion shipped beside the trusted application.
// It never searches PATH or accepts a document-supplied location.
func HelperPath(executable string) string {
	name := "whaleisle-preview"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(executable), name)
}
