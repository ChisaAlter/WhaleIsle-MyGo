package main

import (
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/settings"
)

// TestWallpaperSourcePath pins which value the window backdrop paints. The
// seeded fixture field (goWallpaperPath / configPath) must survive an empty Host
// appearance: the Host theme arrives with wallpaperImage "" on every profile
// that never set one, and preferring that empty value dropped the seeded image
// as soon as the harness connected.
func TestWallpaperSourcePath(t *testing.T) {
	const cfgPath = `C:\tmp\wall.png`
	const hostData = "data:image/png;base64,AAAA"
	for _, tc := range []struct {
		name       string
		hostAccept bool
		hostImage  string
		configPath string
		want       string
	}{
		{"host appearance not accepted yet", false, "", cfgPath, cfgPath},
		{"host accepted but empty: keep the seeded config", true, "", cfgPath, cfgPath},
		{"host accepted with whitespace: keep the seeded config", true, "   ", cfgPath, cfgPath},
		{"host wallpaper wins", true, hostData, cfgPath, hostData},
		{"host only", true, hostData, "", hostData},
		{"nothing set", true, "", "", ""},
		{"trimmed config path", false, "", "  " + cfgPath + "  ", cfgPath},
	} {
		if got := wallpaperSourcePath(tc.hostAccept, tc.hostImage, tc.configPath); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestWallpaperPainterUsesAcceptedHostBackground(t *testing.T) {
	t.Cleanup(func() { chrome.ConfigureBackdrop("", wallpapers.Defaults()) })
	for _, localNone := range []bool{false, true} {
		name := "missing local preference"
		if localNone {
			name = "saved local none before Host adoption"
		}
		t.Run(name, func(t *testing.T) {
			cfg, err := config.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if localNone {
				if err := cfg.SaveWallpaperBackground(wallpapers.Defaults()); err != nil {
					t.Fatal(err)
				}
			}
			prefs := wallpapers.Defaults()
			prefs.Effect, prefs.EffectExplicit = "gradient", true
			s := &mainState{cfgStore: cfg, hostAppearanceAccepted: true, wallpaperControls: &settings.WallpaperEditor{Prefs: prefs}}
			for _, effect := range []string{"gradient", "none", "gradient"} {
				s.wallpaperControls.Prefs.Effect = effect
				s.wallpaperPainter("", true, true)
				if got, want := chrome.BackdropActive(), effect == "gradient"; got != want {
					t.Fatalf("accepted Host %s painted backdrop active=%v", effect, got)
				}
			}
			if cfg.HasWallpaperBackground() != localNone {
				t.Fatal("rendering changed local appearance storage")
			}
		})
	}
}

func TestWallpaperPainterKeepsOfflineFlatFallback(t *testing.T) {
	t.Cleanup(func() { chrome.ConfigureBackdrop("", wallpapers.Defaults()) })
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg}
	s.wallpaperPainter("", true, true)
	if chrome.BackdropActive() || cfg.HasWallpaperBackground() {
		t.Fatal("offline first run seeded a backdrop")
	}
	prefs := wallpapers.Defaults()
	prefs.EffectExplicit = true
	if err := cfg.SaveWallpaperBackground(prefs); err != nil {
		t.Fatal(err)
	}
	s.wallpaperControls = nil
	s.wallpaperPainter("", true, true)
	if chrome.BackdropActive() || s.wallpaperControls.Prefs.Effect != "none" || !s.wallpaperControls.Prefs.EffectExplicit {
		t.Fatal("offline saved none was not preserved")
	}
}
