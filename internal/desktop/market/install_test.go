package market

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallGithubSpec_invalidSpec(t *testing.T) {
	r := InstallGithubSpec("not-a-spec", nil, InstallOptions{})
	if r.OK || r.Error == "" {
		t.Fatalf("result = %+v", r)
	}
}

func TestInstallGithubSpec_notReady(t *testing.T) {
	t.Setenv("WHALEISLE_USER_DATA", t.TempDir())
	r := InstallGithubSpec("github:acme/widget", nil, InstallOptions{NodeBin: filepath.Join(t.TempDir(), "missing-node")})
	if r.OK {
		t.Fatalf("result = %+v", r)
	}
}

func TestInstallReady_withFakeToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("WHALEISLE_USER_DATA", home)
	vendor := t.TempDir()
	root := filepath.Join(vendor, "deepseek-harness")
	bin := filepath.Join(root, "apps", "cli", "lib")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "bin.js"), []byte("// stub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WHALEISLE_VENDOR", vendor)
	node := fakeNode(t)
	if !InstallReady(node) {
		t.Fatal("InstallReady = false, want true with fake toolchain")
	}
}

func fakeNode(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		p := filepath.Join(t.TempDir(), "node.exe")
		if err := os.WriteFile(p, []byte{}, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	p := filepath.Join(t.TempDir(), "node")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}
