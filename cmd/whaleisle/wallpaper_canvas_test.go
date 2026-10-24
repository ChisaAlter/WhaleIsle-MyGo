package main

import (
	"testing"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
)

// TestWallpaperCanvasFill pins the canvas mask: only the dark, image-less,
// non-gradient canvas is masked; light, explicit gradients and wallpapers keep
// the raw theme background so panels and text are not double-darkened.
func TestWallpaperCanvasFill(t *testing.T) {
	base := chrome.Color("#151517")
	flat := wallpapers.Defaults()
	gradient := wallpapers.Defaults()
	gradient.Effect = "gradient"
	gradient.EffectExplicit = true
	gradient.Preset = "dune"
	for _, tc := range []struct {
		name   string
		dark   bool
		path   string
		prefs  wallpapers.Preferences
		masked bool
	}{
		{"dark flat canvas is masked", true, "", flat, true},
		{"light canvas unchanged", false, "", flat, false},
		{"explicit gradient bypasses the mask", true, "", gradient, false},
		{"explicit gradient with no colours bypasses", true, "", func() wallpapers.Preferences {
			p := gradient
			p.Colors = [7]string{}
			return p
		}(), false},
		{"wallpaper keeps the raw base", true, `C:\tmp\wall.png`, flat, false},
	} {
		got := wallpaperCanvasFill(base, tc.dark, tc.path, tc.prefs)
		isMasked := got.R <= 13 && got.G <= 13 && got.B <= 14
		if isMasked != tc.masked {
			t.Fatalf("%s: %v (masked=%v)", tc.name, got, isMasked)
		}
		if !tc.masked && got != base {
			t.Fatalf("%s: base changed to %v", tc.name, got)
		}
	}
}
