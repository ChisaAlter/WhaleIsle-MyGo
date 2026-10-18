//go:build !windows

package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func acquireImportLock(home string) (func(), error) {
	path := filepath.Join(home, ".mygo-import.lock")
	if err := privatePath(path); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("another import or recovery is active: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
