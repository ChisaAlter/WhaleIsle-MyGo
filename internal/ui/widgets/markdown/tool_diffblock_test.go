package markdown

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func diffTestLabels() DiffLabels {
	return DiffLabels{Expand: func(n int) string { return fmt.Sprintf("… %d more lines", n) }, ExpandAria: func(n int) string { return fmt.Sprintf("Expand %d more diff lines", n) }, Collapse: "Collapse", CollapseAria: "Collapse diff"}
}
func diffTestView(t *testing.T, rows []DiffLine, raw string, width int) *ui.Tester {
	t.Helper()
	before := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { theme.SetPreferences(before) })
	st := Style{FontSize: 14, FG: ui.Hex("#0f1115"), Muted: ui.Hex("#61666b"), CodeBG: ui.Hex("#f9fafb"), ToolCodeLanguage: ui.Hex("#81858c")}
	palette := DiffPalette{Added: ui.Hex("#22c55e"), Deleted: ui.Hex("#ec1313"), AddedBG: ui.Hex("#22c55e").Alpha(20.0 / 255), DeletedBG: ui.Hex("#ec1313").Alpha(20.0 / 255)}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { ToolDiffBlock(c, "go", rows, raw, 9, st, palette, diffTestLabels()).Label("diff-card") })
	}, width, 850)
	tt.SetTime(time.Unix(11900, 0))
	tt.Frame()
	return tt
}
func TestToolDiffBlockFoldAndWholeCopy(t *testing.T) {
	rows := []DiffLine{{Kind: "path", Text: "src/a.go"}}
	parts := []string{"src/a.go"}
	for i := 0; i < 13; i++ {
		text := fmt.Sprintf("created-%02d", i)
		rows = append(rows, DiffLine{Kind: "add", Text: text})
		parts = append(parts, "+ "+text)
	}
	raw := strings.Join(parts, "\n")
	tt := diffTestView(t, rows, raw, 616)
	for _, want := range []string{"src/a.go", "+ created-03", "+ created-09", "Expand 5 more diff lines"} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("missing%s%v", want, tt.Texts())
		}
	}
	if tt.HasText("created-04") {
		t.Fatal("middle is not folded")
	}
	card, _ := tt.Find("diff-card")
	if card.H != 258 {
		t.Fatalf("folded%v", card)
	}
	if err := tt.Click("Copy code"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != raw {
		t.Fatalf("copy%q", tt.Clipboard())
	}
	tt.SetClipboard("guard")
	tt.Key(0, ui.KeyEnter)
	if tt.Clipboard() != "guard" {
		t.Fatal("copyguard")
	}
	tt.SetTime(time.Unix(11902, 0))
	tt.Key(0, ui.KeySpace)
	if tt.Clipboard() != raw {
		t.Fatal("copyexpiry")
	}
	if err := tt.Click("Expand 5 more diff lines"); err != nil {
		t.Fatal(err)
	}
	card, _ = tt.Find("diff-card")
	if card.H != 353 || !tt.HasText("created-04") {
		t.Fatalf("expanded%v%v", card, tt.Texts())
	}
	tt.Key(0, ui.KeySpace)
	if tt.HasText("created-04") {
		t.Fatal("Spacefold")
	}
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("created-04") {
		t.Fatal("Enterexpand")
	}
}
func TestToolDiffBlockGridExtentWrapAndSourceScroll(t *testing.T) {
	rows := []DiffLine{{"path", "src/wide.go"}, {"del", strings.Repeat("wide", 120)}, {"add", "new"}, {"context", "tail"}}
	tt := diffTestView(t, rows, "exact", 340)
	initial, _ := tt.Find("diff-card")
	added, _ := tt.Find("+ new")
	context, _ := tt.Find("  tail")
	banner, _ := tt.Find("Wrap lines")
	// Find returns viewport-clipped bounds; full-row fill and its moving stripe
	// are visible evidence of the underlying max-content grid extent.
	if added.W != 340 || context.W != added.W {
		t.Fatalf("diff rows did not fill viewport%v%v", added, context)
	}
	leftBefore := tt.Image().RGBAAt(1, int(added.Y+4))
	rightBefore := tt.Image().RGBAAt(336, int(added.Y+4))
	tt.Scroll(80, 95, 200, 0)
	tt.Frame()
	leftAfter := tt.Image().RGBAAt(1, int(added.Y+4))
	rightAfter := tt.Image().RGBAAt(336, int(added.Y+4))
	still, _ := tt.Find("Wrap lines")
	if leftAfter == leftBefore || rightAfter != rightBefore || still != banner {
		t.Fatalf("source stripe/fill scroll %v/%v ->%v/%v banner%v", leftBefore, rightBefore, leftAfter, rightAfter, still)
	}
	if err := tt.Click("Wrap lines"); err != nil {
		t.Fatal(err)
	}
	wrapped, _ := tt.Find("diff-card")
	added, _ = tt.Find("+ new")
	if wrapped.H <= initial.H || added.W != 340 {
		t.Fatalf("wrap%v%v", wrapped, added)
	}
	tt.Key(0, ui.KeyEnter)
	unwrapped, _ := tt.Find("diff-card")
	if unwrapped.H != initial.H {
		t.Fatalf("unwraprestore%v", unwrapped)
	}
}
func TestToolDiffPathLanguageCanonicalAndFallback(t *testing.T) {
	for _, tc := range []struct {
		paths []string
		want  string
	}{{[]string{"src/main.TS", "other.tsx"}, "typescript"}, {[]string{`C:\dir\a.PS1`, "b.psd1"}, "powershell"}, {[]string{"a.ts", "a.js"}, ""}, {[]string{".env"}, "dotenv"}, {[]string{"dir.ext/noSuffix"}, ""}, {[]string{"a.constructor"}, ""}, {[]string{"a.lock", "b.pem"}, ""}, {nil, ""}} {
		if got := ToolDiffLanguage(tc.paths); got != tc.want {
			t.Fatalf("%q=>%qwant%q", tc.paths, got, tc.want)
		}
	}
}
func TestToolDiffBlockBlankAddedRowKeepsMinimumHeight(t *testing.T) {
	tt := diffTestView(t, []DiffLine{{"path", ""}, {"add", ""}}, "\n+ ", 616)
	card, _ := tt.Find("diff-card")
	row, _ := tt.Find("+ ")
	if card.H != 106 || row.H != 19 {
		t.Fatalf("blank geometry%v%v", card, row)
	}
}
