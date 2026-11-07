//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/windows"
	"whaleisle/internal/paths"
)

// Chromium's Windows ProcessSingleton holds the profile's lockfile open for
// writing. Use that same file rather than inventing a second cooperative lock:
// an Electron owner rejects our open, and our held handle rejects its open.
// When the Electron desktop is running side-by-side on the shared profile,
// fall back to the native sub-profile lock so WhaleIsle-MyGo can still launch
// directly on its own loopback port (DefaultGoPort).
func claimDesktopProfile() (func(), error) {
	release, err := claimDesktopProfileAt(paths.UserDataDir())
	if err == nil {
		return release, nil
	}
	return claimDesktopProfileAt(filepath.Join(paths.UserDataDir(), "native"))
}

func claimDesktopProfileAt(profile string) (func(), error) {
	if err := os.MkdirAll(profile, 0700); err != nil {
		return nil, err
	}
	name, err := windows.UTF16PtrFromString(filepath.Join(profile, "lockfile"))
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", lifecycleText(
			"此数据目录正由另一个鲸屿桌面占用，或无法取得独占访问。请先关闭原版/其他桌面，或使用独立数据目录。",
			"This profile is in use by another Whale Isle desktop, or exclusive access is unavailable. Close the other desktop or choose an independent data directory."), err)
	}
	var once sync.Once
	return func() { once.Do(func() { _ = windows.CloseHandle(h) }) }, nil
}
