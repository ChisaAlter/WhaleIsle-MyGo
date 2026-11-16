package installctrl

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type profileMutexEntry struct {
	mu   sync.Mutex
	refs int
}

var (
	profileLocksMu sync.Mutex
	profileLocks   = map[string]*profileMutexEntry{}
)

func canonicalProfileKey(profileDir string) string {
	p := strings.TrimSpace(profileDir)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

// lockProfile serializes install, allowBuilds write, rollback, and removal
// operations targeting the same profile directory.
func lockProfile(profileDir string) func() {
	key := canonicalProfileKey(profileDir)
	if key == "" {
		return func() {}
	}
	profileLocksMu.Lock()
	entry := profileLocks[key]
	if entry == nil {
		entry = &profileMutexEntry{}
		profileLocks[key] = entry
	}
	entry.refs++
	profileLocksMu.Unlock()

	entry.mu.Lock()
	var once sync.Once
	return func() {
		once.Do(func() {
			entry.mu.Unlock()
			profileLocksMu.Lock()
			entry.refs--
			if entry.refs == 0 && profileLocks[key] == entry {
				delete(profileLocks, key)
			}
			profileLocksMu.Unlock()
		})
	}
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
