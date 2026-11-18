//go:build !windows

package sshremote

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/credentials"
)

func TestSharedTunnelReadsWithoutUnsafeMigrationAndRejectsSave(t *testing.T) {
	dir := t.TempDir()
	file := ConfigPath(dir)
	original := []byte(`{"host":"old.example.test","user":"fixture","password":"synthetic-password"}`)
	if err := os.WriteFile(file, original, 0600); err != nil {
		t.Fatal(err)
	}
	config, err := LoadSharedProfileTunnelConfig(dir)
	if err != nil || config.Host != "old.example.test" || config.Password != "synthetic-password" {
		t.Fatal("shared write restriction blocked existing tunnel read", err)
	}
	config.Host = "new.example.test"
	if err := SaveSharedProfileTunnelConfig(dir, config); !errors.Is(err, credentials.ErrSharedProfileWriteUnavailable) {
		t.Fatal("shared save did not require write ownership", err)
	}
	data, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatal("denied migration or save changed the existing tunnel file")
	}
	if _, err := os.Stat(filepath.Join(dir, credentials.FileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("denied migration created credential storage")
	}
	private := t.TempDir()
	if err := SaveTunnelConfig(private, TunnelConfig{Host: "private.example.test", UseAgent: true}); err != nil {
		t.Fatal("private config contract changed", err)
	}
}
