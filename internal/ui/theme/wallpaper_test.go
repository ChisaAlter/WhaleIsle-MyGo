package theme

import "testing"

func TestNormalizeWallpaperPath(t *testing.T) {
	if got := NormalizeWallpaperPath("  C:\\pics\\bg.png  "); got != `C:\pics\bg.png` {
		t.Fatalf("trim = %q", got)
	}
	if NormalizeWallpaperPath("\t") != "" {
		t.Fatal("whitespace should be empty")
	}
}

func TestWindowWallpaperPath(t *testing.T) {
	if _, ok := WindowWallpaperPath(""); ok {
		t.Fatal("empty path should not be ok")
	}
	path, ok := WindowWallpaperPath("/tmp/wall.jpg")
	if !ok || path != "/tmp/wall.jpg" {
		t.Fatalf("got %q, %v", path, ok)
	}
}
