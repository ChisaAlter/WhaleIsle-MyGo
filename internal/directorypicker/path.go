// Package directorypicker distinguishes filesystem paths from the host's virtual volume root.
package directorypicker

import "strings"

// SelectablePath excludes the virtual Windows volume chooser from workspace
// selection and directory creation. This does not probe or enumerate any disk.
func SelectablePath(path string) bool {
	path = strings.TrimSpace(path)
	normalized := strings.TrimRight(strings.ReplaceAll(path, "/", `\`), `\`)
	return path != "" && !strings.EqualFold(normalized, `\\.\dsh-computer`)
}
