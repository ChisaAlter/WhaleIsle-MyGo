package sshremote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveTunnelConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := TunnelConfig{Host: "example.com", User: "alice", Port: 2222, IdentityFile: "id_ed25519"}
	if err := SaveTunnelConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTunnelConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != cfg.Host || got.User != cfg.User || got.Port != cfg.Port {
		t.Fatalf("%+v", got)
	}
	path := ConfigPath(dir)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	_ = filepath.Base(path)
}

func TestLoadTunnelConfigMissing(t *testing.T) {
	got, err := LoadTunnelConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got.Port != 22 {
		t.Fatalf("port=%d", got.Port)
	}
}
