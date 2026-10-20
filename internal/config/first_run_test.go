package config

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/ui/theme"
)

// TestFirstRunStartsDark pins the shipped first-run appearance: a profile with
// no config.json opens on the dark palette, not on the OS scheme.
func TestFirstRunStartsDark(t *testing.T) {
	if DefaultTheme != theme.FirstRunThemeID {
		t.Fatalf("DefaultTheme = %q, want theme.FirstRunThemeID %q", DefaultTheme, theme.FirstRunThemeID)
	}
	d := Defaults()
	if d.Theme != theme.IDDeepseekDark {
		t.Fatalf("first-run theme = %q, want the dark palette", d.Theme)
	}
	resolved := theme.ResolveWithSystem(d.Theme, false)
	if resolved.Scheme != theme.SchemeDark {
		t.Fatalf("first-run theme followed the system scheme: %q", resolved.Scheme)
	}
	light := theme.ResolveWithSystem(d.Theme, true).Active(true)
	dark := theme.Resolve(d.Theme).Dark
	if light.BG != dark.BG {
		t.Fatalf("first-run palette is not the dark one: %q vs %q", light.BG, dark.BG)
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
	defaults := theme.FirstRunAppearance()
	if appearance.Effect == defaults.Effect && appearance.Colors == defaults.Colors {
		t.Fatal("a profile that chose no backdrop got the first-run backdrop")
	}
}
