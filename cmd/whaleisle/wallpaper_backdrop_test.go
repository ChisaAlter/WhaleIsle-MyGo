package main

import (
	"testing"

	"whaleisle/internal/desktop/wallpapers"
)

// TestWallpaperBackdropPrefs pins the no-image rule: an empty resolved path
// (config empty AND Host empty) must paint the flat theme base, while a seeded
// config image or a Host data URL keeps the stored ambient effect.
func TestWallpaperBackdropPrefs(t *testing.T) {
	explicit := wallpapers.Defaults()
	explicit.Effect = "gradient"
	explicit.EffectExplicit = true
	explicit.Preset = "dune"
	base := wallpapers.Defaults()
	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{"config empty and Host empty: flat base", "", "none"},
		{"whitespace path: flat base", "   ", "none"},
		{"seeded image keeps the aurora", `C:\tmp\wall.png`, "gradient"},
		{"Host data URL keeps the aurora", "data:image/png;base64,AAAA", "gradient"},
	} {
		prefs := base
		if tc.want == "gradient" {
			prefs = explicit
		}
		if got := wallpaperBackdropPrefs(tc.path, prefs).Effect; got != tc.want {
			t.Fatalf("%s: effect %q, want %q", tc.name, got, tc.want)
		}
	}
	// Host empty behaves exactly as empty: source resolution returns "", which
	// is then flattened, while a non-empty Host value still wins.
	if path := wallpaperSourcePath(true, "", ""); wallpaperBackdropPrefs(path, base).Effect != "none" {
		t.Fatal("empty Host must resolve to the flat base")
	}
	if path := wallpaperSourcePath(true, "data:image/png;base64,AAAA", ""); wallpaperBackdropPrefs(path, explicit).Effect != "gradient" {
		t.Fatal("non-empty Host must keep the image effect")
	}
}
