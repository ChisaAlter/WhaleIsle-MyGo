package wallpapers

import (
	"math"
	"testing"
)

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

func TestNormalizeEffectParametersAndDefaults(t *testing.T) {
	d := Defaults()
	if !d.HideSidebarMask || d.Speed != 190 || d.Count != 5 || d.Variant != "orbs" || d.Preset != "default" {
		t.Fatalf("unexpected Defaults(): %+v", d)
	}
	if got := ClampEffectSpeed(0); got != 20 {
		t.Fatalf("ClampEffectSpeed(0) = %v, want 20", got)
	}
	if got := ClampEffectSpeed(360); got != 300 {
		t.Fatalf("ClampEffectSpeed(360) = %v, want 300", got)
	}
	if got := ClampEffectSpeed(math.NaN()); got != 190 {
		t.Fatalf("ClampEffectSpeed(NaN) = %v, want 190", got)
	}
	if got := ClampEffectSpeed(194.6); got != 195 {
		t.Fatalf("ClampEffectSpeed(194.6) = %v, want 195", got)
	}
	if got := ClampEffectCount(0); got != 1 {
		t.Fatalf("ClampEffectCount(0) = %v, want 1", got)
	}
	if got := ClampEffectCount(9); got != 5 {
		t.Fatalf("ClampEffectCount(9) = %v, want 5", got)
	}
	if got := ClampEffectCount(math.Inf(1)); got != 5 {
		t.Fatalf("ClampEffectCount(+Inf) = %v, want 5", got)
	}
	for _, tc := range []struct{ in, want string }{
		{"orbs", "orbs"},
		{"aurora", "aurora"},
		{"chaos", "chaos"},
		{"rays", "rays"},
		{"unknown", "orbs"},
		{"", "orbs"},
	} {
		if got := NormalizeEffectVariant(tc.in); got != tc.want {
			t.Fatalf("NormalizeEffectVariant(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, tc := range []struct{ in, want string }{
		{"default", "default"},
		{"aurora", "aurora"},
		{"sunset", "sunset"},
		{"ocean", "ocean"},
		{"sakura", "sakura"},
		{"dune", "dune"},
		{"custom", "custom"},
		{"", "default"},
		{"bogus", "default"},
	} {
		if got := NormalizeEffectPreset(tc.in); got != tc.want {
			t.Fatalf("NormalizeEffectPreset(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
