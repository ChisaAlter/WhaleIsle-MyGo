package widgets

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestTextShimmerShowsItsText(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() {
			TextShimmer(c, TextShimmerOptions{Text: "Thinking"})
		})
	}, 300, 60)
	if !tt.HasText("Thinking") {
		t.Fatalf("the label is missing: %q", tt.Texts())
	}
}

func TestTextShimmerStillWhenInactive(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() {
			TextShimmer(c, TextShimmerOptions{Text: "Thinking"})
		})
	}, 300, 60)
	base := time.Unix(0, 0)
	tt.SetTime(base)
	first := capture(tt.Image())
	tt.SetTime(base.Add(900 * time.Millisecond))
	if second := capture(tt.Image()); !sameFrame(first, second) {
		t.Fatal("an inactive label moves")
	}
}

func TestTextShimmerSweepsWhenActive(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() {
			TextShimmer(c, TextShimmerOptions{Text: "Thinking about it", Active: true})
		})
	}, 300, 60)
	base := time.Unix(0, 0)
	tt.SetTime(base)
	first := capture(tt.Image())
	// Half a second in the sweep, the highlight crosses the middle.
	tt.SetTime(base.Add(900 * time.Millisecond))
	second := capture(tt.Image())
	if sameFrame(first, second) {
		t.Fatal("the running label does not shimmer")
	}
}

func TestTextShimmerStillWithReduceMotion(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() {
			TextShimmer(c, TextShimmerOptions{Text: "Thinking about it", Active: true})
		})
	}, 300, 60)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	base := time.Unix(0, 0)
	tt.SetTime(base)
	first := capture(tt.Image())
	tt.SetTime(base.Add(900 * time.Millisecond))
	if second := capture(tt.Image()); !sameFrame(first, second) {
		t.Fatal("the label shimmers with reduce motion")
	}
}

func TestTextShimmerCutsLongTextToItsRoom(t *testing.T) {
	tt, theme := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Row(c).Width(60).Children(func() {
				TextShimmer(c, TextShimmerOptions{Text: "a label far too long for sixty DIPs of room"})
			})
		})
	}, 200, 40)
	if x := inkRightOf(tt.Image(), 62, rgb(theme.Background)); x >= 0 {
		t.Fatalf("the label paints past its room, at x=%d", x)
	}
}
