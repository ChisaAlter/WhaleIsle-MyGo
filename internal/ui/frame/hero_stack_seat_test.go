package frame

import (
	"math"
	"testing"
)

// TestHeroStackKeepsTheNativeSeat pins round245: growing the hero card to the
// native 155.34 (round244b) pushed the centred stack up by 20.67 until a fixed
// 41.34 block was added above it. The harness renders at 1440x920, so the
// 1440x860 UIA seats (slogan 314.67, preset 373.33, card 409.33) appear 30 DIP
// lower here; the card height itself is height-independent.
func TestHeroStackKeepsTheNativeSeat(t *testing.T) {
	tt := chatChromeTester(t)
	find := func(name string) (float32, float32) {
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		return r.Y, r.H
	}
	close := func(name string, got, want float32) {
		if math.Abs(float64(got-want)) > 1 {
			t.Fatalf("%s = %.2f, want %.2f ±1", name, got, want)
		}
	}
	sloganY, _ := find("探索未至之境")
	presetY, presetH := find("Composer hero preset row")
	cardY, cardH := find("Composer card")
	close("slogan y", sloganY, 344.67)
	close("preset row y", presetY, 403.33)
	close("preset row height", presetH, 28)
	close("card top", cardY, 439.33)
	close("card height", cardH, 155.34)
	if gap := cardY - (presetY + presetH); math.Abs(float64(gap-8)) > 1 {
		t.Fatalf("preset row to card gap = %.2f, want 8", gap)
	}
}
