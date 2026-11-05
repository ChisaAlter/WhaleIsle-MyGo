//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestDesktopProfileGuardExcludesNativeProfileWriter(t *testing.T) {
	profile := t.TempDir()
	lock := filepath.Join(profile, "lockfile")
	if err := os.WriteFile(lock, []byte("preserve fixture marker"), 0600); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(lock)
	peer, err := windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_READ, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	if release, err := claimDesktopProfileAt(profile); err == nil {
		release()
		t.Fatal("accepted an existing native writer")
	}
	_ = windows.CloseHandle(peer)
	release, err := claimDesktopProfileAt(profile)
	if err != nil {
		t.Fatal(err)
	}
	if peer, err := windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_READ, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0); err == nil {
		windows.CloseHandle(peer)
		release()
		t.Fatal("a native writer bypassed the desktop guard")
	}
	release()
	release()
	if b, err := os.ReadFile(lock); err != nil || string(b) != "preserve fixture marker" {
		t.Fatal("lock content was truncated or deleted")
	}
}
