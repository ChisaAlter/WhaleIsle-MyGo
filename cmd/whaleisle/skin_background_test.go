package main

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestSkinCodexBackdropIsOpaqueAndRestoresSavedSettings(t *testing.T) {
	original := wallpapers.Defaults()
	original.Effect, original.EffectExplicit = "gradient", true
	original.Glass, original.Terminal = 35, 42
	original.Transparent, original.HideSidebarMask = true, true
	originalPath := "saved-wallpaper.png"
	path, p := skinBackdrop(theme.SkinCodex, originalPath, original)
	if path != "" || p.Effect != "none" || p.Glass != 100 || p.Terminal != 100 || p.Transparent || p.HideSidebarMask != original.HideSidebarMask {
		t.Fatal(path, p)
	}
	chrome.ConfigureBackdrop(path, p)
	t.Cleanup(func() { chrome.ConfigureBackdrop("", wallpapers.Defaults()) })
	if chrome.BackdropActive() {
		t.Fatal("Codex left a backdrop active")
	}
	ui.NewTester(func(c *ui.Context) {
		if got := chrome.BackdropSurface(ui.Hex("#181818")); got.A != 255 {
			t.Fatal("Codex surface not opaque", got)
		}
	}, 100, 80)
	for _, skin := range []string{theme.SkinDefault, theme.SkinClaude} {
		gotPath, got := skinBackdrop(skin, originalPath, original)
		if gotPath != originalPath || !reflect.DeepEqual(got, original) {
			t.Fatal("skin mutated saved backdrop", skin, got)
		}
	}
}

func TestSkinCodexFlatCanvasUsesResolvedHalf(t *testing.T) {
	old := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(old) })
	p := theme.Defaults()
	p.Light, p.Dark = "codex", "claude"
	theme.SetPreferences(p)
	s := mainState{themeID: theme.IDDeepseekLight}
	if s.wallpaperCanvasFlatFor("") {
		t.Fatal("Codex requested a transparent masked canvas")
	}
	s.themeID = theme.IDDeepseekDark
	if s.activeSkin() != theme.SkinClaude {
		t.Fatal("pinned dark ignored selected half")
	}
}
