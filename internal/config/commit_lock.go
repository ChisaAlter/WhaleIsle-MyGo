//go:build windows

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// acquireCommitLock takes an exclusive advisory lock on a lockfile that sits
// next to config.json. Every Store in this process — and any other process
// that cooperates by taking the same lock before writing — serializes its
// final compare-and-replace here, so the byte-level baseline check and the
// atomic rename happen with no competing writer in between.
//
// The lock is advisory only: a writer that does not open the lockfile (for
// example the Electron shell editing config.json directly) is not blocked,
// which is why checkBaseline still runs under the lock to reject the commit.
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
	var overlap windows.Overlapped
	if err = windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlap); err != nil {
		file.Close()
		return nil, fmt.Errorf("config: commit lock busy: %w", err)
	}
	return func() {
		_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlap)
		_ = file.Close()
	}, nil
}
