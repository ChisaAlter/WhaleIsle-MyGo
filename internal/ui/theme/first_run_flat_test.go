package theme

import (
	"encoding/json"
	"testing"
)

// TestFirstRunFlatBackdrop: a first-run profile has no wallpaper and no explicit
// background effect, so it must paint the flat theme base (the native profile
// only stores preference: dark).
func TestFirstRunFlatBackdrop(t *testing.T) {
	p := FirstRunAppearance()
	if p.Effect != "none" || p.EffectExplicit {
		t.Fatalf("first run = %q/%v, want none/false", p.Effect, p.EffectExplicit)
	}
}

// TestHostBackgroundExplicitness: the Host seed must stay unset, a stored
// backgroundEffect (even "none") is explicit, and the seeded gradient never
// appears without the key.
func TestHostBackgroundExplicitness(t *testing.T) {
	empty, err := ReadHostSettings(json.RawMessage(`{"preference":"dark"}`))
	if err != nil {
		t.Fatal(err)
	}
	if empty.BackgroundEffect != "" || empty.BackgroundEffectExplicit {
		t.Fatalf("seed effect = %q/%v, want empty/false", empty.BackgroundEffect, empty.BackgroundEffectExplicit)
	}
	if got := empty.Background().Effect; got != "none" {
		t.Fatalf("seed background effect = %q, want none", got)
	}
	gradient, err := ReadHostSettings(json.RawMessage(`{"backgroundEffect":"gradient","backgroundEffectPreset":"dune"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !gradient.BackgroundEffectExplicit || gradient.Background().Effect != "gradient" {
		t.Fatalf("explicit gradient lost: %q/%v", gradient.BackgroundEffect, gradient.BackgroundEffectExplicit)
	}
	none, err := ReadHostSettings(json.RawMessage(`{"backgroundEffect":"none"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !none.BackgroundEffectExplicit || none.Background().Effect != "none" {
		t.Fatalf("explicit none lost: %q/%v", none.BackgroundEffect, none.BackgroundEffectExplicit)
	}
}
