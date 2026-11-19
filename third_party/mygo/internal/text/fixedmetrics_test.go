//go:build !release

package text

import "testing"

// TestFixedMetricsMeasureHalfAnEm pins what the switch promises: a real metric,
// not a zero. Half an em per rune and the style's own line height let a seat
// assertion still catch a composition that grew a row or a margin.
func TestFixedMetricsMeasureHalfAnEm(t *testing.T) {
	s := newSystem()
	s.UseFixedMetrics()
	p := Params{Text: "abcd", Style: Style{Family: "Inter", Size: 20, LineHeight: 1.25}}
	l := s.Layout(p)
	if len(l.Lines) != 1 {
		t.Fatalf("lines: %+v", l.Lines)
	}
	if got := l.Lines[0].Width; got != 40 {
		t.Fatalf("width %v, want 4 runes at half of 20", got)
	}
	if got := l.Lines[0].Height; got != 25 {
		t.Fatalf("line height %v, want 1.25em of 20", got)
	}
	// The family is a request the fixed engine ignores: that is the pinning.
	other := Params{Text: "abcd", Style: Style{Family: "No Such Family", Size: 20, LineHeight: 1.25}}
	if got := s.Layout(other).Lines[0]; got.Width != l.Lines[0].Width || got.Height != l.Lines[0].Height {
		t.Fatalf("a family changed the fixed metrics: %+v against %+v", got, l.Lines[0])
	}
	// Chinese takes the same half em, which is what the boot fixture needs: no
	// host fallback decides whether 鲸屿 fits on one line.
	wide := Params{Text: "鲸屿", Style: Style{Size: 20}}
	if got := s.Layout(wide).Lines[0].Width; got != 20 {
		t.Fatalf("CJK width %v, want two runes at half of 20", got)
	}
}
