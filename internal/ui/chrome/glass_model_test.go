package chrome

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
)

func glassPrefs(glass float64) wallpapers.Preferences {
	p := wallpapers.Defaults()
	p.Glass = glass
	return p
}

// TestGlassCompositeNativeModel reconstructs the pass303 native samples with the
// solved model (panel token at glassOpacity over the desktop pixel): each
// back-solved desktop must recompose to the measured native value within ±2.
func TestGlassCompositeNativeModel(t *testing.T) {
	const alpha = float32(0.6) // glassOpacity 60
	for _, tc := range []struct {
		name    string
		panel   string
		desktop string
		want    [3]int
	}{
		{"sidebar", "#1b1b1c", "#05142e", [3]int{18, 24, 35}},
		{"hero", "#151517", "#354049", [3]int{34, 38, 43}},
	} {
		got := GlassComposite(Color(tc.panel), Color(tc.desktop), alpha)
		if absInt(int(got.R)-tc.want[0]) > 2 || absInt(int(got.G)-tc.want[1]) > 2 || absInt(int(got.B)-tc.want[2]) > 2 {
			t.Fatalf("%s: composite %v, want %v", tc.name, got, tc.want)
		}
	}
	if got := ShowThroughAlpha(glassPrefs(60), true); got != alpha {
		t.Fatalf("show-through alpha = %v, want %v", got, alpha)
	}
	if got := ShowThroughAlpha(glassPrefs(60), false); got != 1 {
		t.Fatalf("light alpha = %v, want 1", got)
	}
	if got := ShowThroughAlpha(glassPrefs(100), true); got != 1 {
		t.Fatalf("fully opaque glass alpha = %v, want 1", got)
	}
	transparent := glassPrefs(60)
	transparent.Transparent = true
	if got := ShowThroughAlpha(transparent, true); got != 1 {
		t.Fatalf("transparent theme alpha = %v, want 1", got)
	}
}

// TestGlassSurfaceAlphaOverDesktop renders a dark glass surface with no
// wallpaper and checks the painted alpha equals glassOpacity, so the window
// (transparent) composites the desktop through it as the native model does.
func TestGlassSurfaceAlphaOverDesktop(t *testing.T) {
	prefs := glassPrefs(60)
	ConfigureBackdrop("", prefs)
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	tt2 := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Column(c).Fill().Children(func() {
			ui.Box(c).Fill().Draw(Glass(ui.RGBA(27, 27, 28, 255), MaterialSidebar, 0))
		})
	}, 40, 20)
	start := time.Unix(1000, 0)
	tt2.SetTime(start)
	tt2.SetTime(start.Add(2 * time.Second))
	// The tester root paints the dark base (21,21,23); the 60% surface blends
	// to ~(24,24,25), which is the composite the transparent window shows over
	// a desktop once the OS composites it.
	got := tt2.Image().RGBAAt(20, 10)
	if got.R < 23 || got.R > 26 {
		t.Fatalf("glass composite = %v, want ~(24,24,25)", got)
	}
}
