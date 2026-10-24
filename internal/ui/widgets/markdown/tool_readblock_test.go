package markdown

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func readTestLabels() ReadLabels {
	return ReadLabels{Window: func(n, total int) string { return fmt.Sprintf("Showing %d of %d lines", n, total) }, Expand: func(n int) string { return fmt.Sprintf("… %d more lines", n) }, ExpandAria: func(n int) string { return fmt.Sprintf("Expand %d more lines", n) }, Collapse: "Collapse", CollapseAria: "Collapse content"}
}
func readTestView(t *testing.T, lines []ReadLine, total, width int, lang string) *ui.Tester {
	t.Helper()
	before := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { theme.SetPreferences(before) })
	st := Style{FontSize: 14, FG: ui.RGB(20, 20, 20), Muted: ui.RGB(100, 100, 100), CodeBG: ui.RGB(249, 250, 251)}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ToolReadBlock(c, "src/read.go", lang, lines, total, 8, st, readTestLabels()).Label("read-card")
		})
	}, width, 850)
	tt.SetTime(time.Unix(11800, 0))
	tt.Frame()
	return tt
}
func TestToolReadFoldKeepsSourceNumbersAndCopiesWholeWindow(t *testing.T) {
	lines := make([]ReadLine, 14)
	parts := make([]string, 14)
	for i := range lines {
		parts[i] = fmt.Sprintf("returned-%02d", i)
		lines[i] = ReadLine{Number: 101 + i*2, Text: parts[i]}
	}
	lines[0].Text = "\tfirst\r"
	parts[0] = lines[0].Text
	lines[13].Text = ""
	parts[13] = ""
	tt := readTestView(t, lines, 180, 700, "not-a-language")
	for _, visible := range []string{"101", "107", "121", "127", "returned-03", "returned-10", "Showing 14 of 180 lines"} {
		if _, ok := tt.Find(visible); !ok {
			t.Fatalf("missing%s %v", visible, tt.Texts())
		}
	}
	if _, ok := tt.Find("returned-04"); ok {
		t.Fatal("folded middle visible")
	}
	if err := tt.Click("Copy code"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != strings.Join(parts, "\n") {
		t.Fatalf("copy%q", tt.Clipboard())
	}
	if _, ok := tt.Find("Copied"); !ok {
		t.Fatal("no copied state")
	}
	tt.SetTime(time.Unix(11801, 1000000))
	if err := tt.Click("Expand 6 more lines"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("returned-04"); !ok {
		t.Fatal("middle absent after expand")
	}
	if _, ok := tt.Find("Collapse content"); !ok {
		t.Fatal("collapse action absent")
	}
	tt.Key(0, ui.KeySpace)
	if _, ok := tt.Find("returned-04"); ok {
		t.Fatal("Space did not fold")
	}
	tt.Key(0, ui.KeyEnter)
	if _, ok := tt.Find("returned-04"); !ok {
		t.Fatal("Enter did not unfold")
	}
	if err := tt.Click("Copy code"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != strings.Join(parts, "\n") {
		t.Fatal("expanded copy changed")
	}
}
func TestToolReadEmptyAndBlankFilesKeepDistinctCopyBehavior(t *testing.T) {
	empty := readTestView(t, []ReadLine{}, 0, 500, "")
	if _, ok := empty.Find("Copy code"); ok {
		t.Fatal("empty file exposed copy")
	}
	if _, ok := empty.Find("Wrap lines"); !ok {
		t.Fatal("empty file lost wrapping")
	}
	r, _ := empty.Find("read-card")
	if r.H != 68 {
		t.Fatalf("empty height%v", r)
	}
	blank := readTestView(t, []ReadLine{{Number: 1, Text: ""}}, 1, 500, "")
	if err := blank.Click("Copy code"); err != nil {
		t.Fatal(err)
	}
	r, _ = blank.Find("read-card")
	if r.H != 87 {
		t.Fatalf("blank line minheight%v", r)
	}
}
func TestToolReadWrapGutterAndColumnScrolling(t *testing.T) {
	lines := []ReadLine{{Number: 1000, Text: strings.Repeat("column", 80)}, {Number: 1014, Text: "tail"}}
	tt := readTestView(t, lines, 1100, 400, "go")
	before, _ := tt.Find("read-card")
	n, _ := tt.Find("1014")
	tail, _ := tt.Find("tail")
	if tail.X-(n.X+n.W) < 11.9 || tail.X-(n.X+n.W) > 12.1 {
		t.Fatalf("gutter gap%v %v", n, tail)
	}
	if err := tt.Click("Wrap lines"); err != nil {
		t.Fatal(err)
	}
	after, _ := tt.Find("read-card")
	if after.H <= before.H {
		t.Fatalf("wrap%v ->%v", before, after)
	}
	n, _ = tt.Find("1014")
	tail, _ = tt.Find("tail")
	if n.Y != tail.Y {
		t.Fatalf("wrapped gutter baseline%v %v", n, tail)
	}
	tt.Key(0, ui.KeyEnter)
	restored, _ := tt.Find("read-card")
	if restored.H != before.H {
		t.Fatalf("unwrapped changed%v vs%v", restored, before)
	}
	beforeTail, _ := tt.Find("tail")
	beforeBanner, _ := tt.Find("Wrap lines")
	tt.Scroll(80, 70, 240, 0)
	tt.Frame()
	afterTail, tailVisible := tt.Find("tail")
	afterBanner, _ := tt.Find("Wrap lines")
	if tailVisible && afterTail.X >= beforeTail.X {
		t.Fatalf("horizontal scroll did not move source %v -> %v", beforeTail, afterTail)
	}
	if afterBanner != beforeBanner {
		t.Fatalf("horizontal scroll moved toolbar %v -> %v", beforeBanner, afterBanner)
	}
}
func TestToolReadHighlightsWholeWindowAndPreservesLineContent(t *testing.T) {
	raw := "/* head\ncomment */ const x = 42;\n\treturn '尾';\r\n"
	rows := toolReadHighlightedLines(toolGrammarForHint("JS"), raw, 4, Style{FG: ui.RGB(20, 20, 20)}, 11)
	parts := strings.Split(raw, "\n")
	for i, row := range rows {
		var text strings.Builder
		for _, span := range row {
			text.WriteString(span.Text)
		}
		if text.String() != parts[i] {
			t.Fatalf("line%d%q vs%q", i, text.String(), parts[i])
		}
	}
	if len(rows[1]) < 2 {
		t.Fatal("whole-window grammar lost comment end")
	}
	if toolGrammarForHint("constructor") != "" || toolGrammarForHint(" go") != "" || toolGrammarForHint("JS") != "typescript" {
		t.Fatal("installed language allowlist")
	}
}
