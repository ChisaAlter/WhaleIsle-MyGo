//go:build windows

package credentials

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

func sharedProfileWriteSupport() error { return nil }

func acquirePrivateProfileWrite(dir string) (func(), error) { return acquireProfileWrite(dir) }

var heldDesktopProfiles = struct {
	sync.Mutex
	paths map[string]*desktopProfileLease
}{paths: make(map[string]*desktopProfileLease)}

type desktopProfileLease struct {
	handle windows.Handle
	refs   int
}

func profileReleaser(key string, lease *desktopProfileLease) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			heldDesktopProfiles.Lock()
			defer heldDesktopProfiles.Unlock()
			lease.refs--
			if lease.refs == 0 {
				delete(heldDesktopProfiles.paths, key)
				_ = windows.CloseHandle(lease.handle)
			}
		})
	}
}

func profileKey(dir string) string {
	path, err := filepath.Abs(dir)
	if err != nil {
		path = filepath.Clean(dir)
	}
	if canonical, err := filepath.EvalSymlinks(path); err == nil {
		path = canonical
	}
	return strings.ToLower(path)
}

// ClaimDesktopProfileWriter owns Chromium's no-share profile lockfile. The
// launcher uses this same handle for its lifetime; credential commits can then
// recognize their own lock rather than treating it as another application's.
func ClaimDesktopProfileWriter(dir string) (func(), error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	name, err := windows.UTF16PtrFromString(filepath.Join(dir, "lockfile"))
	if err != nil {
		return nil, err
	}
	heldDesktopProfiles.Lock()
	defer heldDesktopProfiles.Unlock()
	h, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSharedProfileBusy, err)
	}
	key := profileKey(dir)
	lease := &desktopProfileLease{handle: h, refs: 1}
	heldDesktopProfiles.paths[key] = lease
	return profileReleaser(key, lease), nil
}

func acquireProfileWrite(dir string) (func(), error) {
	key := profileKey(dir)
	heldDesktopProfiles.Lock()
	if lease := heldDesktopProfiles.paths[key]; lease != nil {
		lease.refs++
		heldDesktopProfiles.Unlock()
		return profileReleaser(key, lease), nil
	}
	heldDesktopProfiles.Unlock()
	return ClaimDesktopProfileWriter(dir)
}
