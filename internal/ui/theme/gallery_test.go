package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListWallpapers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.png"), []byte{1}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ListWallpapers(dir, filepath.Join(t.TempDir(), "missing"))
	if len(got) != 1 || filepath.Base(got[0]) != "a.png" {
		t.Fatalf("got %v", got)
	}
}

func TestWallpaperGalleryDirs(t *testing.T) {
	dirs := WallpaperGalleryDirs("C:\\data", "C:\\app")
	if len(dirs) < 2 {
		t.Fatalf("dirs = %v", dirs)
	}
}
