package text

import (
	"math"
	"testing"
)

// TestSpanPadXAdvance checks a span's pad is part of the layout, not just
// paint: the glyph after a padded span starts PadX*2 further right than
// the same glyph without the pad.
func TestSpanPadXAdvance(t *testing.T) {
	s := newSystem()
	const pad = float32(5)
	x := func(spans string) float32 {
		l := s.Layout(Params{Text: "abcdefgh", Style: Style{Size: 16}, Spans: spans})
		for _, line := range l.Lines {
			for _, g := range line.Glyphs {
				if g.Cluster == 6 { // g, after the padded span
					return line.X + g.X
				}
			}
		}
		t.Fatal("no glyph for g")
		return 0
	}
	plain := x("")
	padded := x(EncodeSpans([]Span{{End: 3}, {End: 6, PadX: pad}, {End: 8}}))
	if d := padded - plain; math.Abs(float64(d-2*pad)) > 0.01 {
		t.Errorf("a pad of %v moved the following glyph %v, not %v", pad, d, 2*pad)
	}
}

// TestSpanPadXWraps checks a wrapped line never starts with a dangling
// pad: pad glyphs cover no runes, so no line may begin with one.
func TestSpanPadXWraps(t *testing.T) {
	s := newSystem()
	l := s.Layout(Params{
		Text:  "一段inline行内文字继续往后排列更多行内容",
		Style: Style{Size: 16},
		Width: 120,
		Spans: EncodeSpans([]Span{{End: 2}, {End: 8, PadX: 5}, {End: 23}}),
	})
	if len(l.Lines) < 2 {
		t.Fatalf("expected wrapping, got %d line", len(l.Lines))
	}
	for i := range l.Lines {
		if g := l.Lines[i].Glyphs; len(g) > 0 && g[0].Runes <= 0 {
			t.Errorf("line %d starts with a dangling pad", i)
		}
	}
	// The pad's glyph keeps exactly the pad's width and reports the span's
	// edge rune, so it groups with the span for painting.
	var found bool
	for i := range l.Lines {
		for _, g := range l.Lines[i].Glyphs {
			if g.Runes == 0 && g.Cluster == 2 {
				found = true
				if g.Advance != 5 {
					t.Errorf("pad glyph's advance is %v, not 5", g.Advance)
				}
			}
		}
	}
	if !found {
		t.Error("no pad glyph at the span's start")
	}
}
