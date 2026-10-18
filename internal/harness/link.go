package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// EnsureDirectoryLink makes target a directory junction (Windows) or
// symlink (elsewhere) pointing at source. It refuses to replace a real
// directory that is not already a link. Returns true when a new link was
// created.
func EnsureDirectoryLink(source, target string) (bool, error) {
	sourceReal, err := filepath.EvalSymlinks(source)
	if err != nil {
		return false, fmt.Errorf("plugin source %s: %w", source, err)
	}
	info, err := os.Stat(sourceReal)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("plugin source is not a directory: %s", sourceReal)
	}
	if directoryLinkMatches(sourceReal, target) {
		return false, nil
	}
	if li, err := os.Lstat(target); err == nil {
		if !isReplaceableLink(target, li) {
			return false, fmt.Errorf("refusing to replace unknown content at %s", target)
		}
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return false, err
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}
	if err := symlinkDir(sourceReal, target); err != nil {
		if os.IsExist(err) && directoryLinkMatches(sourceReal, target) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// isReplaceableLink reports whether target is a symlink or Windows junction
// that we may remove and recreate. mklink /J junctions often appear as
// ModeIrregular (not ModeSymlink) on Windows; os.Readlink still works.
func isReplaceableLink(target string, li os.FileInfo) bool {
	if li.Mode()&os.ModeSymlink != 0 {
		return true
	}
	if _, err := os.Readlink(target); err == nil {
		return true
	}
	return false
}

func directoryLinkMatches(source, target string) bool {
	actual, err := os.Readlink(target)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(actual) {
		actual = filepath.Join(filepath.Dir(target), actual)
	}
	return sameFile(source, actual)
}

func sameFile(a, b string) bool {
	ra, err1 := filepath.Abs(a)
	rb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return equalFoldPath(ra, rb)
	}
	return ra == rb
}

func equalFoldPath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
