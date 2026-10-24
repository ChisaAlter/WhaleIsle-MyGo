package theme

import (
	"reflect"
	"testing"

	"whaleisle/internal/desktop/wallpapers"
)

// TestFirstRunAppearanceMatchesHarnessDefaults pins the install defaults of
// ui-theme/src/theme-settings.ts DEFAULT_THEME_SETTINGS, so a fresh profile
// matches the harness profile: no wallpaper and no explicit background effect,
// so the backdrop is the flat theme base (the native profile only stores
// preference: dark); the dune preset stays ready for an explicit gradient.
func TestFirstRunAppearanceMatchesHarnessDefaults(t *testing.T) {
	p := FirstRunAppearance()
	if p.Effect != "none" || p.Preset != "dune" || p.Variant != "orbs" {
		t.Fatalf("first-run backdrop = %q/%q/%q, want none/dune/orbs", p.Effect, p.Preset, p.Variant)
	}
	if p.Speed != 190 || p.Count != 5 {
		t.Fatalf("first-run ambient speed/count = %v/%v, want 190/5", p.Speed, p.Count)
	}
	if p.Colors != ([7]string{"#D1D6DC", "#C6C4DB", "#F4F4F8", "#B7D4C8", "#B2D0CB", "#D0D4DD", "#CED5DD"}) {
		t.Fatalf("first-run gradient colors drift: %v", p.Colors)
	}
	if p.Colors != wallpapers.PresetColors("dune") {
		t.Fatalf("first-run colors disagree with the dune preset: %v", p.Colors)
	}
	if p.Glass != 80 || p.Terminal != 75 {
		t.Fatalf("first-run glass/terminal = %v/%v, want 80/75", p.Glass, p.Terminal)
	}
	if !p.HideSidebarMask || p.Transparent {
		t.Fatalf("first-run mask/transparent = %v/%v, want hidden and opaque", p.HideSidebarMask, p.Transparent)
	}
	if p.Blur != 0 || p.Pixelate != 0 {
		t.Fatalf("first-run wallpaper blur/pixelate = %v/%v, want the harness 0", p.Blur, p.Pixelate)
	}
	if p.CursorEnabled || p.MetallicButtons {
		t.Fatalf("first-run pointer effects should stay off: enabled=%v metallic=%v", p.CursorEnabled, p.MetallicButtons)
	}
	if p.CursorEffect != "trail" || p.CursorSpeed != 100 || p.CursorSize != 100 {
		t.Fatalf("first-run pointer scheme drift: %q %v %v", p.CursorEffect, p.CursorSpeed, p.CursorSize)
	}
}

func TestFirstRunAppearanceLeavesNoWallpaper(t *testing.T) {
	p := FirstRunAppearance()
	if p.Original != "" || p.Crop != wallpapers.DefaultCrop() {
		t.Fatalf("first-run appearance should carry no wallpaper: %q %+v", p.Original, p.Crop)
	}
	// The ambient backdrop is the default effect, not a one-off mutation of the
	// shared defaults other callers read.
	if again := FirstRunAppearance(); !reflect.DeepEqual(again, p) {
		t.Fatal("FirstRunAppearance is not stable across calls")
	}
}

// TestFirstRunThemeFollowsTheOS pins the first-run theme id: the harness
// install default is preference "system" (ui-theme/src/theme-settings.ts
// DEFAULT_PREFERENCE), so a fresh launch follows the OS scheme instead of
// forcing a palette the user did not choose.
func TestFirstRunThemeFollowsTheOS(t *testing.T) {
	if FirstRunThemeID != DefaultID {
		t.Fatalf("FirstRunThemeID = %q, want %q", FirstRunThemeID, DefaultID)
	}
	fresh := ResolveWithSystem(FirstRunThemeID, false)
	if fresh.Scheme != SchemeLight {
		t.Fatalf("first-run light scheme = %q, want %q", fresh.Scheme, SchemeLight)
	}
	if got := fresh.Active(false).BG; got != deepseekLight.BG {
		t.Fatalf("first-run light background = %q, want %q", got, deepseekLight.BG)
	}
	dark := ResolveWithSystem(FirstRunThemeID, true)
	if dark.Scheme != SchemeDark {
		t.Fatalf("first-run dark scheme = %q, want %q", dark.Scheme, SchemeDark)
	}
	if got := dark.Active(true).BG; got != deepseekDark.BG {
		t.Fatalf("first-run dark background = %q, want %q", got, deepseekDark.BG)
	}
}
