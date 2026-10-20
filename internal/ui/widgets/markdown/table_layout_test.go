package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A CJK cell measured by rune count used to be half its real width, so a six
// character column wrapped. The table now measures the text and keeps the cell
// on one line while the column stays inside its 40% share.
func TestTableCJKColumnKeepsCellOnOneLine(t *testing.T) {
	source := "| 现象 | 根因 |\n|---|---|\n| EACCES | 父目录不存在或无法读取 |"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			View(c, source, Style{FontSize: 14, FG: ui.RGB(0, 0, 0), Muted: ui.RGB(90, 90, 90), AvailableWidth: 748})
		})
	}, 760, 300)
	cell, ok := tt.Find("父目录不存在或无法读取")
	if !ok {
		t.Fatalf("CJK cell missing: %q", tt.Texts())
	}
	if got, want := cell.H, BodyLineHeight(Style{FontSize: 14}); got > want+.5 {
		t.Fatalf("CJK cell wrapped: height %v, want one %v line", got, want)
	}
	// 11 CJK glyphs at 14 DIP plus 2×10 padding, not the old rune count.
	if cell.W < 11*13 {
		t.Fatalf("column width %v does not follow the measured text", cell.W)
	}
}

// The same cell inside a narrow axis must wrap at the 40% ceiling instead of
// pushing the table past its container without a scroll rule.
func TestTableNarrowAxisWrapsAtColumnShare(t *testing.T) {
	source := "| 现象 | 根因 |\n|---|---|\n| EACCES | 父目录不存在或无法读取 |"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			View(c, source, Style{FontSize: 14, FG: ui.RGB(0, 0, 0), Muted: ui.RGB(90, 90, 90), AvailableWidth: 240})
		})
	}, 300, 400)
	cell, ok := tt.Find("父目录不存在或无法读取")
	if !ok {
		t.Fatalf("CJK cell missing: %q", tt.Texts())
	}
	// 40% of 240 is 96; with 2×10 padding the text may use ~76, so it wraps.
	if cell.W > 96 {
		t.Fatalf("narrow column %v exceeded the 40%% share of 240", cell.W)
	}
	if cell.H <= BodyLineHeight(Style{FontSize: 14}) {
		t.Fatalf("cell at the column ceiling did not wrap: %v", cell.H)
	}
}

// A table that fits must not reserve a scrollbar; one that does not fit keeps
// every cell reachable through the scroll wrapper.
func TestTableScrollsOnlyWhenItExceedsTheAxis(t *testing.T) {
	narrow := "| 一 | 二 |\n|---|---|\n| a | b |"
	wide := "| 列一 | 列二 | 列三 | 列四 |\n|---|---|---|---|\n| " + strings.Repeat("很长的内容", 12) + " | x | y | z |"
	fits := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			View(c, narrow, Style{FontSize: 14, FG: ui.RGB(0, 0, 0), Muted: ui.RGB(90, 90, 90), AvailableWidth: 600})
		})
	}, 700, 200)
	scrolled := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			View(c, wide, Style{FontSize: 14, FG: ui.RGB(0, 0, 0), Muted: ui.RGB(90, 90, 90), AvailableWidth: 300})
		})
	}, 700, 400)
	for _, label := range []string{"一", "二", "a", "b"} {
		if _, ok := fits.Find(label); !ok {
			t.Fatalf("fitting table lost cell %q", label)
		}
	}
	for _, label := range []string{"列一", "列二", "列三", "列四", "x", "y", "z"} {
		if _, ok := scrolled.Find(label); !ok {
			t.Fatalf("wide table lost cell %q", label)
		}
	}
	// The oversized first column is clipped to 40% of 300 and wraps, but the
	// remaining columns still fit inside the scrolled band.
	if cell, ok := scrolled.Find("列四"); !ok || cell.X > 300 {
		t.Fatalf("trailing column not laid out inside the scroll band: %+v", cell)
	}
}
