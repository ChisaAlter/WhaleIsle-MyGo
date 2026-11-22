//go:build windows

package paths

import "golang.org/x/sys/windows"

// osDocumentsDir returns the user's Documents folder through the known-folder
// API, which follows OneDrive and other redirections the profile path does
// not.
func osDocumentsDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
}
