//go:build !windows

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// acquireCommitLock takes an exclusive advisory lock on a lockfile that sits
// next to config.json. See the windows variant for the full contract; the
// byte-level baseline check still runs under the lock so a non-cooperating
// writer is rejected rather than overwritten.
func acquireCommitLock(configPath string) (func(), error) {
	dir := filepath.Dir(configPath)
	// The data dir may not exist yet on a first run; writeAtomic creates it
	// too, but the lock must be opened before that step runs.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, ".config.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("config: commit lock busy: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
