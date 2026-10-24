package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/text"
)

func TestElementBreakAllWrapping(t *testing.T) {
	const raw = "alpha bravocharlie,delta/echo?x=y; "
	style := text.Style{Family: "monospace", Size: 16}
	width := textSystem().Layout(text.Params{Text: "alpha bra", Style: style, KeepSpaces: true}).Width + 0.25
	var normal, all, before, after *node
	NewTester(func(c *Context) {
		Column(c).Width(width).AlignItems(Start).Children(func() {
			normal = Text(c, strings.Repeat(raw, 3)).Font("monospace").FontSize(16).Width(width).node()
			all = Text(c, strings.Repeat(raw, 3)).Font("monospace").FontSize(16).Width(width).BreakAll().node()
			before = Text(c, raw).Font("monospace").FontSize(16).Width(width).NoWrap().BreakAll().node()
			after = Text(c, raw).Font("monospace").FontSize(16).Width(width).BreakAll().NoWrap().node()
		})
	}, 400, 600)
	if normal.tl == nil || all.tl == nil || len(normal.tl.Lines) < 3 || len(all.tl.Lines) < 3 {
		t.Fatal("headless public API did not produce wrapped layouts")
	}
	if normal.tl.Lines[0].End != len("alpha ") || all.tl.Lines[0].End <= normal.tl.Lines[0].End {
		t.Fatalf("public BreakAll still prefers words: normal %d, all %d", normal.tl.Lines[0].End, all.tl.Lines[0].End)
	}
	if !all.tl.Params.BreakAll || all.text != strings.Repeat(raw, 3) || all.tl.Params.Text != all.text {
		t.Error("BreakAll flag or raw text did not reach the text system")
	}
	for _, e := range []*node{before, after} {
		if len(e.tl.Lines) != 1 || e.tl.Width <= width {
			t.Errorf("NoWrap lost precedence: %d lines, width %g", len(e.tl.Lines), e.tl.Width)
		}
	}
	_, normalMin := normal.leafWidths()
	_, allMin := all.leafWidths()
	if allMin >= normalMin || allMin >= width {
		t.Errorf("BreakAll kept a whole-word minimum: normal %g, all %g, box %g", normalMin, allMin, width)
	}
}

func TestElementBreakAllKeepsGraphemesAndCopyText(t *testing.T) {
	const raw = "alpha e\u0301👩‍👩‍👧‍👦🇺🇳👍🏽✈️,bravo/charlie?x=y; delta"
	var paragraph *node
	width := float32(1)
	tt := NewTester(func(c *Context) {
		paragraph = Text(c, raw).FontSize(18).Width(width).BreakAll().Selectable().node()
	}, 400, 600)
	var b text.Boundaries
	b.Reset([]rune(raw))
	boundaries := b.GraphemeOffsets()
	for i, line := range paragraph.tl.Lines {
		if !slices.Contains(boundaries, line.Start) || !slices.Contains(boundaries, line.End) {
			t.Errorf("public BreakAll line %d splits a grapheme: %d-%d", i, line.Start, line.End)
		}
	}
	if paragraph.text != raw || paragraph.st.editor.source != raw || string(paragraph.tl.Runes) != raw {
		t.Fatal("wrapping changed source or selection text")
	}
	// Select across several visual lines using normal hit testing.
	width = 80
	tt.Frame()
	from, to := len([]rune("alpha ")), len([]rune(raw))-len([]rune(" delta"))
	selectBetween(t, tt, raw, from, raw, to)
	tt.Key(Cmd, KeyC)
	if want := string([]rune(raw)[from:to]); tt.Clipboard() != want {
		t.Fatalf("copy across wrapped lines = %q, want %q", tt.Clipboard(), want)
	}
	tt.Key(Cmd, KeyA)
	tt.Command("copy")
	if tt.Clipboard() != raw || !tt.HasText(raw) {
		t.Fatalf("Select All changed raw text: copied %q", tt.Clipboard())
	}
}
