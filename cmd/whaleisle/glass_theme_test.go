package main

import (
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/renderpolicy"
)

// useGlassTheme turns the optional wallpaper-and-glass theme on for a test
// that checks what it paints.
func useGlassTheme(t *testing.T) {
	t.Helper()
	renderpolicy.SetGlass(true)
	t.Cleanup(func() { renderpolicy.SetGlass(false) })
}

func TestGlassThemeMigrationKeepsOnlyAnExplicitWallpaper(t *testing.T) {
	on, off := true, false
	for _, tc := range []struct {
		cfg  config.Config
		want bool
	}{
		{config.Config{}, false},
		{config.Config{WallpaperPath: "C:/wallpapers/sea.jpg"}, true},
		{config.Config{WallpaperPath: "C:/wallpapers/sea.jpg", GlassTheme: &off}, false},
		{config.Config{GlassTheme: &on}, true},
	} {
		if got := tc.cfg.GlassThemeOn(); got != tc.want {
			t.Errorf("%+v: glass theme %v, want %v", tc.cfg, got, tc.want)
		}
	}
}

func TestGlassThemeOffPaintsNoBackdrop(t *testing.T) {
	renderpolicy.SetGlass(false)
	s := &mainState{cfgStore: minimalModeStore(t)}
	_ = s.wallpaperPainter("C:/wallpapers/sea.jpg", true)
	if s.wallpaperCanvasFlatFor("C:/wallpapers/sea.jpg") != true {
		t.Fatal("glass theme off kept a wallpaper canvas")
	}
}
