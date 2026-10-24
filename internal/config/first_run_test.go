package config

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/ui/theme"
)

// TestFirstRunFollowsTheSystem pins the shipped first-run appearance: a profile
// with no config.json follows the OS scheme, matching the harness install
// default (theme settings preference "system").
func TestFirstRunFollowsTheSystem(t *testing.T) {
	if DefaultTheme != theme.FirstRunThemeID {
		t.Fatalf("DefaultTheme = %q, want theme.FirstRunThemeID %q", DefaultTheme, theme.FirstRunThemeID)
	}
	d := Defaults()
	if d.Theme != theme.DefaultID {
		t.Fatalf("first-run theme = %q, want the system-following family %q", d.Theme, theme.DefaultID)
	}
	light := theme.ResolveWithSystem(d.Theme, false)
	if light.Scheme != theme.SchemeLight {
		t.Fatalf("first-run theme did not follow the light system scheme: %q", light.Scheme)
	}
	dark := theme.ResolveWithSystem(d.Theme, true)
	if dark.Scheme != theme.SchemeDark {
		t.Fatalf("first-run theme did not follow the dark system scheme: %q", dark.Scheme)
	}
	if light.Active(false).BG == dark.Active(true).BG {
		t.Fatalf("system schemes resolved to the same palette: %q", light.Active(false).BG)
	}
}

func TestIsFirstRunFollowsTheFile(t *testing.T) {
	dir := t.TempDir()
	fresh, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.IsFirstRun() {
		t.Fatal("a directory without config.json is a first run")
	}
	// Saving during the first session keeps the flag: the choices were defaulted
	// from it.
	if err := fresh.Update(func(c *Config) { c.Locale = "en" }); err != nil {
		t.Fatal(err)
	}
	if !fresh.IsFirstRun() {
		t.Fatal("saving within the first run cleared its flag")
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.IsFirstRun() {
		t.Fatal("an existing config.json is not a first run")
	}
}

// TestSavedAppearanceSurvivesAFreshOpen checks that first-run defaults never
// override a profile that already stored appearance settings.
func TestSavedAppearanceSurvivesAFreshOpen(t *testing.T) {
	dir := t.TempDir()
	raw := `{"theme":"deepseek.light","goWallpaperPath":"C:/pictures/wall.png",` +
		`"goWallpaperBackground":{"backgroundEffect":"none","glassOpacity":35,"wallpaperBlur":40}}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.IsFirstRun() {
		t.Fatal("a saved config.json was reported as a first run")
	}
	cfg := s.Get()
	if cfg.Theme != "deepseek.light" {
		t.Fatalf("saved theme overridden: %q", cfg.Theme)
	}
	if cfg.WallpaperPath != "C:/pictures/wall.png" {
		t.Fatalf("saved wallpaper overridden: %q", cfg.WallpaperPath)
	}
	appearance, err := s.WallpaperBackground()
	if err != nil {
		t.Fatal(err)
	}
	if appearance.Effect != "none" || appearance.Glass != 35 || appearance.Blur != 40 {
		t.Fatalf("saved appearance overridden: %+v", appearance)
	}
	if !appearance.EffectExplicit {
		t.Fatal("a stored backgroundEffect must stay explicit")
	}
}
