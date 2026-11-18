//go:build windows

package sshremote

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/windows"
	"whaleisle/internal/credentials"
)

func TestSharedTunnelSaveRefusalPreservesConfiguration(t *testing.T) {
	dir := t.TempDir()
	initial := TunnelConfig{Host: "old.example.test", Password: "synthetic-password"}
	if err := SaveSharedProfileTunnelConfig(dir, initial); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(ConfigPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	lock, _ := windows.UTF16PtrFromString(dir + `\lockfile`)
	peer, err := windows.CreateFile(lock, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(peer)
	read, err := LoadSharedProfileTunnelConfig(dir)
	if err != nil || read.Password != initial.Password {
		t.Fatal("peer lock blocked existing encrypted config read", err)
	}
	read.Host, read.Password = "new.example.test", "new-synthetic-password"
	if err := SaveSharedProfileTunnelConfig(dir, read); !errors.Is(err, credentials.ErrSharedProfileBusy) {
		t.Fatal("shared save bypassed peer", err)
	}
	data, err := os.ReadFile(ConfigPath(dir))
	if err != nil || !bytes.Equal(data, original) || read.Password != "new-synthetic-password" {
		t.Fatal("failed shared save lost caller draft or changed persisted config")
	}
}
