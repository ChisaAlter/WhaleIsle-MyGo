package text

import (
	"runtime"
	"slices"
	"strings"
	"testing"
)

// Exercise the native engine (DirectWrite even without a window on Windows)
// and the portable fallback with the same layout and cache assertions.
func breakAllSystems(t *testing.T, check func(*testing.T, *System)) {
	t.Helper()
	for _, fallback := range []bool{false, true} {
		name := "native"
		if fallback {
			name = "fallback"
		}
		t.Run(name, func(t *testing.T) {
			s := newSystem()
			if fallback {
				s.eng = &stubEngine{}
			} else if _, stub := s.engine().(*stubEngine); stub && runtime.GOOS == "windows" {
				t.Fatal("Windows headless text must exercise DirectWrite")
			}
			check(t, s)
		})
	}
}

func TestBreakAllWrapsWithoutWordPreference(t *testing.T) {
	breakAllSystems(t, func(t *testing.T, s *System) {
		style := Style{Family: "monospace", Size: 16}
		p := Params{Text: strings.Repeat("alpha bravocharlie,delta/echo?x=y; ", 5), Style: style}
		p.Width = s.Layout(Params{Text: "alpha bra", Style: style, KeepSpaces: true}).Width + 0.25
		normal := s.Layout(p)
		if len(normal.Lines) < 3 || normal.Lines[0].End != len("alpha ") {
			t.Fatalf("normal wrapping: first line %+v, %d lines", normal.Lines[0], len(normal.Lines))
		}
		p.BreakAll = true
		all := s.Layout(p)
		if all == normal || all.Lines[0].End <= normal.Lines[0].End || all.Lines[0].End >= len("alpha bravocharlie") {
			t.Fatalf("BreakAll did not fill past a word boundary: normal end %d, all end %d", normal.Lines[0].End, all.Lines[0].End)
		}
		if s.Layout(p) != all {
			t.Error("BreakAll layout was not cached")
		}
		p.BreakAll = false
		if s.Layout(p) != normal {
			t.Error("BreakAll replaced the normal layout's cache entry")
		}
		p.BreakAll, p.NoBreakWords = true, true
		if both := s.Layout(p); !slices.EqualFunc(both.Lines, all.Lines, func(a, b Line) bool { return a.Start == b.Start && a.End == b.End }) {
			t.Error("NoBreakWords overrode BreakAll")
		}
		if all.Params.Text != p.Text || string(all.Runes) != p.Text || all.Truncated {
			t.Error("wrapping changed or truncated the raw text")
		}
		for i, line := range all.Lines {
			if line.Width > p.Width+0.5 || i > 0 && line.Start != all.Lines[i-1].End {
				t.Errorf("line %d does not fit or lost text: %+v", i, line)
			}
		}
		if all.Lines[len(all.Lines)-1].End != len(all.Runes) {
			t.Error("last line lost raw text")
		}
	})
}

func TestBreakAllKeepsGraphemes(t *testing.T) {
	breakAllSystems(t, func(t *testing.T, s *System) {
		// Include combining marks, ZWJ emoji, regional indicators, skin tone,
		// variation selectors and an Indic conjunct, all wider than the box.
		const raw = "Ae\u0301👩‍👩‍👧‍👦🇺🇳👍🏽✈️क्\u200dषZ"
		p := Params{Text: raw, Style: Style{Size: 18}, Width: 1, BreakAll: true, KeepSpaces: true}
		l := s.Layout(p)
		var b Boundaries
		b.Reset([]rune(raw))
		boundaries := b.GraphemeOffsets()
		if len(l.Lines) < 3 {
			t.Fatalf("narrow box did not wrap: %d lines", len(l.Lines))
		}
		for i, line := range l.Lines {
			if !slices.Contains(boundaries, line.Start) || !slices.Contains(boundaries, line.End) {
				t.Errorf("line %d splits a grapheme: %d-%d, boundaries %v", i, line.Start, line.End, boundaries)
			}
			if line.End <= line.Start || i > 0 && line.Start != l.Lines[i-1].End {
				t.Errorf("line %d did not progress continuously: %d-%d", i, line.Start, line.End)
			}
		}
		if l.Params.Text != raw || string(l.Runes) != raw || l.Lines[len(l.Lines)-1].End != len(l.Runes) {
			t.Error("grapheme wrapping changed raw text")
		}
	})
}

func TestBreakAllGraphemeRepairKeepsSpans(t *testing.T) {
	s := newSystem()
	if _, stub := s.engine().(*stubEngine); stub {
		t.Skip("fallback has no styled fonts")
	}
	const raw = "A🇺🇳B🇺🇸C"
	l := s.Layout(Params{
		Text: raw, Style: Style{Size: 16}, Width: 1, BreakAll: true,
		Spans: EncodeSpans([]Span{{End: 4, Size: 24}, {End: 7, Size: 32}}),
	})
	// Regional-indicator pairs can require native re-shaping. Both the
	// repaired line and the suffix keep their span styles and rune offsets.
	if len(l.Lines) != 5 {
		t.Fatalf("got %d lines, want one per whole grapheme", len(l.Lines))
	}
	for i, line := range l.Lines {
		if len(line.Glyphs) == 0 {
			t.Errorf("line %d lost its glyphs", i)
		}
		for _, g := range line.Glyphs {
			wantSize := float32(24)
			if g.Cluster >= 4 {
				wantSize = 32
			}
			if g.Cluster < line.Start || g.Cluster+g.Runes > line.End || g.Size != wantSize {
				t.Errorf("line %d (%d-%d) lost span style or rune offsets: %+v, want size %g", i, line.Start, line.End, g, wantSize)
			}
		}
	}
}

func TestBreakAllNoWrapWins(t *testing.T) {
	breakAllSystems(t, func(t *testing.T, s *System) {
		p := Params{Text: "alpha bravocharlie,delta/echo\nsecond paragraph", Style: Style{Size: 16}, Width: 12, NoWrap: true}
		noWrap := s.Layout(p)
		p.BreakAll, p.NoBreakWords = true, true
		all := s.Layout(p)
		if len(all.Lines) != 2 || all.Width <= p.Width || !slices.EqualFunc(noWrap.Lines, all.Lines, func(a, b Line) bool { return a.Start == b.Start && a.End == b.End && a.Width == b.Width }) {
			t.Fatalf("BreakAll overrode NoWrap: %d lines, width %g", len(all.Lines), all.Width)
		}
	})
}
