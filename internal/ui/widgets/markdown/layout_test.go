package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The transcript opts into DocumentLayout; these assertions pin the Original
// MarkdownText ladder that a flat span run used to lose.
func TestDocumentLayoutKeepsBlockLadder(t *testing.T) {
	source := "# Title\n\nBody paragraph\n\n- item one\n- item two\n\n> quoted line\n\n---\n\nAfter rule"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, source, Style{DocumentLayout: true, FontSize: 14, FG: ui.RGB(0, 0, 0), Muted: ui.RGB(90, 90, 90), QuoteInset: 14})
		})
	}, 700, 900)
	heading, ok := tt.Find("Title")
	if !ok {
		t.Fatal("heading not rendered")
	}
	body, ok := tt.Find("Body paragraph")
	if !ok {
		t.Fatal("paragraph not rendered")
	}
	if gap := body.Y - (heading.Y + heading.H); gap < 15.5 {
		t.Fatalf("heading-to-paragraph gap %v, want the 16px block margin", gap)
	}
	item, ok := tt.Find("item one")
	if !ok {
		t.Fatal("list item not rendered")
	}
	if indent := item.X - body.X; indent < 16.5 || indent > 19.5 {
		t.Fatalf("list indent %v, want the Original 18px marker gutter", indent)
	}
	quote, ok := tt.Find("quoted line")
	if !ok {
		t.Fatal("blockquote not rendered")
	}
	if indent := quote.X - body.X; indent < 13 || indent > 15.5 {
		t.Fatalf("quote inset %v, want the Original 14px rule padding", indent)
	}
	if item.Y < body.Y || quote.Y < item.Y {
		t.Fatal("block order changed")
	}
}

func TestTableColumnsFollowContentAndScroll(t *testing.T) {
	source := "| id | description |\n|---|---|\n| 1 | a much longer description cell |"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, source, Style{FontSize: 14, FG: ui.RGB(0, 0, 0), Muted: ui.RGB(90, 90, 90)})
		})
	}, 900, 400)
	idCell, ok := tt.Find("id")
	if !ok {
		t.Fatal("header row missing")
	}
	descCell, ok := tt.Find("description")
	if !ok {
		t.Fatal("second header cell missing")
	}
	valueCell, ok := tt.Find("1")
	if !ok {
		t.Fatal("body row missing")
	}
	if valueCell.Y <= idCell.Y {
		t.Fatal("body row did not follow the header row")
	}
	if descCell.X <= idCell.X {
		t.Fatal("columns did not advance")
	}
	if width := descCell.X - idCell.X; width > 220 {
		t.Fatalf("short column took %v, content-proportional widths expected", width)
	}
}

func TestCodeBlockLineNumbersAndPaletteFollowTheme(t *testing.T) {
	code := "func main() {\n\tprintln(1)\n}"
	st := Style{FontSize: 14, FG: ui.RGB(20, 20, 20), Muted: ui.RGB(90, 90, 90), CodeBG: ui.RGB(245, 245, 245)}
	plain := highlightedCodeSpans("go", code, st)
	numbered := highlightedCodeSpans("go", code, Style{FontSize: 14, FG: st.FG, Muted: st.Muted, CodeBG: st.CodeBG, LineNumbers: true})
	joined := func(spans []ui.Span) string {
		var out strings.Builder
		for _, span := range spans {
			out.WriteString(span.Text)
		}
		return out.String()
	}
	if strings.Contains(joined(plain), " 1  ") {
		t.Fatal("line numbers rendered without opt-in")
	}
	if !strings.Contains(joined(numbered), " 1  ") || !strings.Contains(joined(numbered), " 3  ") {
		t.Fatalf("line gutter missing: %q", joined(numbered))
	}
	if CodeSize(st) != 11 {
		t.Fatalf("code size %v, want the 11px markdown-code-block token", CodeSize(st))
	}
	dark := chromaPalette(true)
	light := chromaPalette(false)
	if dark == nil || light == nil || dark == light {
		t.Fatal("dark and light code palettes are not distinct")
	}
}
