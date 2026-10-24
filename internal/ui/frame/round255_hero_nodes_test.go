package frame

import (
	"math"
	"testing"
)

// TestRound255HeroRowNodes is the round255 diagnostic: it prints every node the
// Hero headline row emits, with its rect, so a live UIA name/size can be matched
// to the frame path that produces it.
//
// It pins the audit facts:
//   - exactly one fish node exists ("Conversation hero fish", 34 x 24.67),
//     2.69 DIP before the slogan;
//   - the badge is the named text "大肥鱼版" (48.67 x 15.33) at the native x 936,
//     14.31 after the slogan — i.e. UIA's 34x24.67 @ 728.67 and the harness'
//     49x15.4 @ 936 are two different nodes of one row, not two badge versions.
func TestRound255HeroRowNodes(t *testing.T) {
	tt := chatChromeTester(t)
	want := []struct {
		name    string
		x, w, h float32
	}{
		{"Conversation hero fish", 728.64, 34, 24.67},
		{"探索未至之境", 765.33, 156.67, 32},
		{"大肥鱼版", 936, 48.67, 15.33},
	}
	got := map[string][4]float32{}
	for _, w := range want {
		r, ok := tt.Find(w.name)
		if !ok {
			t.Fatalf("missing %q in the hero row", w.name)
		}
		t.Logf("hero row node %-24q x=%.2f y=%.2f w=%.2f h=%.2f", w.name, r.X, r.Y, r.W, r.H)
		got[w.name] = [4]float32{r.X, r.Y, r.W, r.H}
		if math.Abs(float64(r.X-w.x)) > 1 || math.Abs(float64(r.W-w.w)) > 1 || math.Abs(float64(r.H-w.h)) > 1 {
			t.Errorf("%q = %.2fx%.2f @ %.2f, want %.2fx%.2f @ %.2f", w.name, r.W, r.H, r.X, w.w, w.h, w.x)
		}
	}
	fish, slogan, badge := got["Conversation hero fish"], got["探索未至之境"], got["大肥鱼版"]
	gapFish := slogan[0] - (fish[0] + fish[2])
	gapBadge := badge[0] - (slogan[0] + slogan[2])
	t.Logf("gaps: fish->slogan=%.2f slogan->badge=%.2f (row centre stays put)", gapFish, gapBadge)
	if math.Abs(float64(gapFish-2.69)) > 0.05 || math.Abs(float64(gapBadge-14.31)) > 0.05 {
		t.Errorf("gaps = %.2f / %.2f, want 2.69 / 14.31", gapFish, gapBadge)
	}
}
