package main

import "testing"

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
