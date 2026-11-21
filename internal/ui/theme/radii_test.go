package theme

import "testing"

// The default deepseek family sets no radius overrides, so its table is the
// base.css token set field for field.
func TestResolvedRadiiDefault(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	SetPreferences(Defaults())
	if got := ResolvedRadii(IDDeepseekLight, false); got != DefaultRadii {
		t.Fatalf("ResolvedRadii = %+v, want %+v", got, DefaultRadii)
	}
}

// The official builtin family carries its own --dsw-radius-* overrides
// (builtin_official.go), and every one of them lands on the resolved table.
func TestResolvedRadiiOfficialFamily(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	p := Defaults()
	p.Light, p.Dark = "codex", "codex"
	SetPreferences(p)
	want := Radii{XS: 4, SM: 6, MD: 8, LG: 12, XL: 14, Panel: 14, Dialog: 16, Compact: 4, Inset: 8, Badge: 8, ButtonSM: 6, Action: 8, Button: 8, Composer: 26, ComposerFrame: 28}
	if got := ResolvedRadii(IDDeepseekLight, false); got != want {
		t.Fatalf("ResolvedRadii = %+v, want %+v", got, want)
	}
}

// A radius override that is not a plain <number>px — a calc() expression, a
// percentage, inherit — is ignored, and the default stands.
func TestResolvedRadiiIgnoresNonPx(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	p := Defaults()
	f := library[0]
	f.ID = "radii-test"
	f.Light.Overrides = map[string]string{"--dsw-radius-md": "calc(1px + 2px)", "--dsw-radius-sm": " 7px ", "--dsw-radius-lg": "50%"}
	p.Custom = []Family{f}
	p.Light, p.Dark = "radii-test", "radii-test"
	SetPreferences(p)
	got := ResolvedRadii(IDDeepseekLight, false)
	if got.MD != DefaultRadii.MD || got.LG != DefaultRadii.LG {
		t.Fatalf("non-px overrides leaked: %+v", got)
	}
	if got.SM != 7 {
		t.Fatalf("trimmed px override not applied: %+v", got)
	}
}

// ResolvedRadii runs for themed elements every frame; the memoized path must
// not allocate.
func TestResolvedRadiiNoAllocation(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	SetPreferences(Defaults())
	ResolvedRadii(IDDeepseekLight, false)
	if allocs := testing.AllocsPerRun(100, func() { ResolvedRadii(IDDeepseekLight, false) }); allocs != 0 {
		t.Fatalf("ResolvedRadii allocated %.2f per call", allocs)
	}
}
