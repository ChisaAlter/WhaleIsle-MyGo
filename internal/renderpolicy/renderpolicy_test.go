package renderpolicy

import (
	"reflect"
	"testing"

	"whaleisle/internal/desktop/wallpapers"
)

func TestMinimalKeepsEveryStoredPreferenceWhenOff(t *testing.T) {
	SetMinimal(false)
	stored := wallpapers.Preferences{
		SkinWallpaperEnabled: true,
		CursorEnabled:        true,
		Effect:               "gradient",
		EffectExplicit:       true,
		Preset:               "aurora",
		Transparent:          true,
		Glass:                35,
		Terminal:             45,
	}
	if got := Background(stored); !reflect.DeepEqual(got, stored) {
		t.Fatalf("policy off changed the preferences: %+v", got)
	}
	if got := Wallpaper("C:/wallpapers/sea.png"); got != "C:/wallpapers/sea.png" {
		t.Fatalf("policy off dropped the wallpaper: %q", got)
	}
}

func TestMinimalPausesEveryEffectWithoutEditingTheStore(t *testing.T) {
	stored := wallpapers.Preferences{
		SkinWallpaperEnabled: true,
		CursorEnabled:        true,
		MetallicButtons:      true,
		Effect:               "gradient",
		EffectExplicit:       true,
		Preset:               "aurora",
		Variant:              "orbs",
		Speed:                240,
		Count:                7,
		Transparent:          true,
		Glass:                35,
		Terminal:             45,
		CursorEffect:         "splash",
	}
	SetMinimal(true)
	defer SetMinimal(false)

	got := Background(stored)
	if got.Effect != "none" || !got.EffectExplicit || got.Preset != "" {
		t.Fatalf("ambient background survived minimal mode: %+v", got)
	}
	if got.SkinWallpaperEnabled || got.Transparent || got.Glass != 100 || got.Terminal != 100 {
		t.Fatalf("materials stayed see-through under minimal mode: %+v", got)
	}
	if got.CursorEnabled {
		t.Fatalf("pointer decoration survived minimal mode: %+v", got)
	}
	if gotPath := Wallpaper("C:/wallpapers/sea.png"); gotPath != "" {
		t.Fatalf("wallpaper survived minimal mode: %q", gotPath)
	}
	if stored.Effect != "gradient" || !stored.Transparent || stored.Glass != 35 || stored.CursorEnabled != true {
		t.Fatalf("projection edited the stored preferences: %+v", stored)
	}
	// Everything the switch does not name is still the user's own choice.
	if got.Variant != "orbs" || got.Speed != 240 || got.Count != 7 || got.CursorEffect != "splash" {
		t.Fatalf("unrelated effect preferences were changed: %+v", got)
	}
	if !got.MetallicButtons {
		t.Fatalf("the button sheen is not one of the paused effects: %+v", got)
	}
}

func TestSetMinimalReportsOnlyRealChanges(t *testing.T) {
	SetMinimal(false)
	if !SetMinimal(true) {
		t.Fatal("turning minimal mode on reported no change")
	}
	if SetMinimal(true) {
		t.Fatal("turning minimal mode on twice reported a change")
	}
	if !SetMinimal(false) {
		t.Fatal("turning minimal mode off reported no change")
	}
}
