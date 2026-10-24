package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOpenToleratesUTF8BOM pins the loader's tolerance for a tool-written BOM:
// a seeded fixture config with a UTF-8 BOM must load, not die with
// `invalid character '\ufeff'` and the "config read failed" dialog.
func TestOpenToleratesUTF8BOM(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"theme":"deepseek.dark","goWallpaperPath":"C:/wall.png"}`)
	data := append([]byte{0xEF, 0xBB, 0xBF}, body...)
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("BOM config rejected: %v", err)
	}
	cfg := store.Get()
	if cfg.Theme != "deepseek.dark" {
		t.Fatalf("theme = %q, want deepseek.dark", cfg.Theme)
	}
	if cfg.WallpaperPath != "C:/wall.png" {
		t.Fatalf("wallpaper = %q, want C:/wall.png", cfg.WallpaperPath)
	}
	// The bytes on disk stay as written (baseline keeps the original file).
	if raw, err := os.ReadFile(filepath.Join(dir, FileName)); err != nil || len(raw) != len(data) {
		t.Fatalf("config file changed: %d bytes, want %d", len(raw), len(data))
	}
}
