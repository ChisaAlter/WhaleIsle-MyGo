package wallpapers

import "testing"

// TestNormalizeEffect pins the backdrop precedence: without an image only an
// explicit backgroundEffect can paint the gradient; an unset/empty value and a
// non-explicit seed fall back to the flat theme base.
func TestNormalizeEffect(t *testing.T) {
	base := Defaults()
	explicit := func(effect string) Preferences {
		p := base
		p.Effect = effect
		p.EffectExplicit = true
		return p
	}
	seededGradient := base
	seededGradient.Effect = "gradient"
	for _, tc := range []struct {
		name string
		path string
		p    Preferences
		want string
	}{
		{"unset: flat", "", base, "none"},
		{"empty effect, not explicit: flat", "", explicit(""), "none"},
		{"gradient without explicitness: flat", "", seededGradient, "none"},
		{"explicit gradient: gradient", "", explicit("gradient"), "gradient"},
		{"explicit none: none", "", explicit("none"), "none"},
		{"seeded image keeps gradient", `C:\tmp\wall.png`, explicit("gradient"), "gradient"},
		{"seeded image with none stays none", `C:\tmp\wall.png`, base, "none"},
		{"host data URL keeps gradient", "data:image/png;base64,AAAA", explicit("gradient"), "gradient"},
	} {
		if got := NormalizeEffect(tc.path, tc.p).Effect; got != tc.want {
			t.Fatalf("%s: effect %q, want %q", tc.name, got, tc.want)
		}
	}
	// The seeded default is explicitly flat now, so nothing turns the aurora on.
	if Defaults().Effect != "none" || Defaults().EffectExplicit {
		t.Fatalf("defaults = %q/%v, want none/false", Defaults().Effect, Defaults().EffectExplicit)
	}
}
