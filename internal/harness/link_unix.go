//go:build !windows

package harness

import "os"

func symlinkDir(source, target string) error {
	return os.Symlink(source, target)
}
