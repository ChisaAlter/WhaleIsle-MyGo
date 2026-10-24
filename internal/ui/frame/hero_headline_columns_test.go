package frame

import (
	"math"
	"testing"
)

// TestHeroHeadlineMatchesTheNativeColumns pins round254: with the narrower badge
// the centred headline row had drifted right (slogan 772.64 vs native 765.33).
// The fix redistributes the same total width — fish 2.69 from the slogan, badge
// 14.31 after it — so the row keeps its centre while the slogan lands on the
// native 765.33 and the badge on 936 (1440-wide UIA).
func TestHeroHeadlineMatchesTheNativeColumns(t *testing.T) {
	tt := chatChromeTester(t)
	find := func(name string) [4]float32 {
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		return [4]float32{r.X, r.Y, r.W, r.H}
	}
	close := func(name string, got, want float32) {
		if math.Abs(float64(got-want)) > 1 {
			t.Fatalf("%s = %.2f, want %.2f ±1", name, got, want)
		}
	}
	slogan := find("探索未至之境")
	badge := find("大肥鱼版")
	fish := find("Conversation hero fish")
	close("slogan x", slogan[0], 765.33)
	close("slogan width", slogan[2], 156.67)
	close("badge x", badge[0], 936)
	close("badge width", badge[2], 48.67)
	close("badge height", badge[3], 15.33)
	// The row is still centred: the fish-to-slogan gap plus the slogan-to-badge
	// gap split the native 17 DIP into 2.69 + 14.31.
	close("fish-to-slogan gap", slogan[0]-(fish[0]+fish[2]), 2.69)
	close("slogan-to-badge gap", badge[0]-(slogan[0]+slogan[2]), 14.31)
}
