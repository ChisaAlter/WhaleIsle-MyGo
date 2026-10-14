//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"path/filepath"
	"testing"
)

// Filesystem-only fixture. No browser, Host, daemon or user HTML is started.
func TestPreviewCleanupOwnsOriginalDirectory(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, "owned")
	if err := os.Mkdir(owned, 0700); err != nil {
		t.Fatal(err)
	}
	identity, err := os.Lstat(owned)
	if err != nil {
		t.Fatal(err)
	}
	p := &previewEnvironment{path: owned, identity: identity}
	replacement := filepath.Join(root, "moved")
	if err := os.Rename(owned, replacement); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(owned, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(owned, "do-not-delete")
	if err := os.WriteFile(marker, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	p.removeOwnedDirectory()
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("removed a replacement directory:", err)
	}
	p.path = replacement
	p.removeOwnedDirectory()
	if _, err := os.Stat(replacement); !os.IsNotExist(err) {
		t.Fatal("owned directory not removed:", err)
	}
}
