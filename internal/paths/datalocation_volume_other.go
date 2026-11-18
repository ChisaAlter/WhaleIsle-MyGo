//go:build !windows

package paths

// volumeIsRemote reports whether path is on a network volume. The check is
// Windows-only: the location chooser's refusal is about mapped and UNC drives,
// and other platforms mount network shares as ordinary paths.
func volumeIsRemote(string) bool { return false }
