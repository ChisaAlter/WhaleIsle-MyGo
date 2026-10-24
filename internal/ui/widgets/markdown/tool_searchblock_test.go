package markdown

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func searchTestLabels() SearchLabels {
	return SearchLabels{PathsSummary: func(shown int, total float64, truncated bool) string {
		if truncated {
			return fmt.Sprintf("Showing %d of %s paths", shown, SearchNumber(total))
		}
		return fmt.Sprintf("%d paths", shown)
	}, MatchesSummary: func(shown int, total float64, files int, truncated bool) string {
		if truncated {
			return fmt.Sprintf("Showing %d of %s matches · %d files", shown, SearchNumber(total), files)
		}
		return fmt.Sprintf("%d matches · %d files", shown, files)
	}, Copy: "Copy", Copied: "Copied", NoResults: "No results", Collapse: "Collapse", CollapseAria: "Collapse results", Expand: func(n int) string { return fmt.Sprintf("… %d more lines", n) }, ExpandAria: func(n int) string { return fmt.Sprintf("Expand %d more result lines", n) }}
}
func searchTestView(t *testing.T, data SearchData, width int) *ui.Tester {
	t.Helper()
	previous := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { theme.SetPreferences(previous) })
	style := Style{FG: ui.Hex("#0f1115"), Muted: ui.Hex("#61666b"), ToolCodeLanguage: ui.Hex("#81858c"), CodeBG: ui.Hex("#f9fafb"), CodeBanner: ui.Hex("#f9fafb")}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { ToolSearchBlock(c, data, 8, style, searchTestLabels()).Label("search-card") })
	}, width, 900)
	tt.SetTime(time.Unix(12000, 0))
	tt.Frame()
	return tt
}
func searchTestFiles() SearchData {
	data := SearchData{Kind: "matches", Truncated: true, Total: 30, Files: []SearchFile{{Path: "src/a.go"}, {Path: "src/b.go"}}}
	for i := 0; i < 4; i++ {
		data.Files[0].Matches = append(data.Files[0].Matches, SearchMatch{LineNumber: float64(i + 1), Line: fmt.Sprintf("first-%02d", i)})
	}
	for i := 0; i < 6; i++ {
		data.Files[1].Matches = append(data.Files[1].Matches, SearchMatch{LineNumber: float64(i + 20), Line: fmt.Sprintf("second-%02d", i)})
	}
	return data
}
func TestToolSearchRowsRestoreTailAndGroupCollapse(t *testing.T) {
	data := searchTestFiles()
	rows := searchRows(data, map[int]bool{})
	head, tail, hidden := searchVisibleRows(rows, 8, false)
	if len(rows) != 12 || len(head) != 4 || len(tail) != 4 || hidden != 4 || tail[0].kind != "file" || tail[0].index != 1 || tail[1].line != "second-03" {
		t.Fatalf("tail attribution%#v%#v hidden%d", head, tail, hidden)
	}
	tt := searchTestView(t, data, 616)
	card, _ := tt.Find("search-card")
	if card.H != 253 {
		t.Fatalf("folded geometry%v", card)
	}
	if !tt.HasText("src/b.go 6") || tt.HasText("second-02") || !tt.HasText("second-03") {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click("src/b.go 6"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("second-03") || tt.HasText("Expand 4 more result lines") || !tt.HasText("first-03") {
		t.Fatal(tt.Texts())
	}
	tt.Key(0, ui.KeySpace)
	if !tt.HasText("Expand 4 more result lines") {
		t.Fatal("Space did not restore group")
	}
	if err := tt.Click("Expand 4 more result lines"); err != nil {
		t.Fatal(err)
	}
	card, _ = tt.Find("search-card")
	if card.H != 341 || !tt.HasText("second-02") {
		t.Fatalf("expanded%v%v", card, tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if tt.HasText("second-02") {
		t.Fatal("Enter did not fold")
	}
	one := data
	one.Files = one.Files[1:]
	rows = searchRows(one, nil)
	_, tail, _ = searchVisibleRows(rows, 4, false)
	if tail[0].kind != "match" {
		t.Fatal("single file duplicated header")
	}
}
func TestToolSearchCopyAllGroupsFeedbackGuardAndEmpty(t *testing.T) {
	data := searchTestFiles()
	tt := searchTestView(t, data, 616)
	want := searchCopyText(data)
	if err := tt.Click("src/a.go 4"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != want || !strings.Contains(want, "src/a.go\n1: first-00\n") || !strings.Contains(want, "\n\nsrc/b.go\n20: second-00") {
		t.Fatal(tt.Clipboard())
	}
	tt.SetClipboard("guard")
	tt.Key(0, ui.KeyEnter)
	if tt.Clipboard() != "guard" {
		t.Fatal("guard failed")
	}
	tt.SetTime(time.Unix(12002, 0))
	tt.Key(0, ui.KeySpace)
	if tt.Clipboard() != want {
		t.Fatal("feedback expired but copy did not reset")
	}
	for _, kind := range []string{"paths", "matches"} {
		empty := searchTestView(t, SearchData{Kind: kind}, 616)
		card, _ := empty.Find("search-card")
		if !empty.HasText("No results") || empty.HasText("Copy") || card.H != 81 {
			t.Fatalf("empty%s%v%v", kind, card, empty.Texts())
		}
	}
	zero := SearchData{Kind: "matches", Files: []SearchFile{{Path: "empty.go"}}}
	tt = searchTestView(t, zero, 616)
	if tt.HasText("No results") || !tt.HasText("empty.go 0") {
		t.Fatal("zero-match file is not a header row")
	}
	if got := searchCopyText(SearchData{Kind: "matches", Files: []SearchFile{{Path: "a", Matches: []SearchMatch{{LineNumber: 1e21, Line: "x"}}}, {Path: ""}}}); got != "a\n1e+21: x\n\n" {
		t.Fatal(got)
	}
}
func TestToolSearchPathsNoWrapScrollAndFixedBanner(t *testing.T) {
	data := SearchData{Kind: "paths", Paths: []string{strings.Repeat("directory/", 90) + "last.go", "short.go"}, Total: 2}
	tt := searchTestView(t, data, 320)
	inkCount := func() int {
		image := tt.Image()
		count := 0
		for y := 68; y < 87; y++ {
			for x := 14; x < 100; x++ {
				p := image.RGBAAt(x, y)
				if p.R < 150 && p.G < 150 && p.B < 150 {
					count++
				}
			}
		}
		return count
	}
	before := inkCount()
	copy, _ := tt.Find("Copy")
	height, _ := tt.Find("search-card")
	if height.H != 102 || tt.HasText("Wrap lines") {
		t.Fatal("search must retain no-wrap rows")
	}
	tt.Scroll(80, 80, 190, 0)
	tt.Frame()
	copyAfter, _ := tt.Find("Copy")
	if before == 0 || inkCount() != 0 || copy != copyAfter {
		t.Fatalf("horizontal scrolling moved banner or did not scroll: ink %d->%d, copy%v->%v", before, inkCount(), copy, copyAfter)
	}
}
func TestToolSearchNumberAndSourceImmutability(t *testing.T) {
	for _, tc := range []struct {
		number float64
		want   string
	}{{0, "0"}, {1, "1"}, {1e20, "100000000000000000000"}, {1e21, "1e+21"}, {1e30, "1e+30"}, {1e300, "1e+300"}, {9007199254740993, "9007199254740992"}} {
		if got := SearchNumber(tc.number); got != tc.want {
			t.Fatalf("%v ->%s expected%s", tc.number, got, tc.want)
		}
	}
	data := searchTestFiles()
	before := fmt.Sprintf("%#v", data)
	collapsed := map[int]bool{1: true}
	rows := searchRows(data, collapsed)
	searchVisibleRows(rows, 4, false)
	searchCopyText(data)
	if fmt.Sprintf("%#v", data) != before || !reflect.DeepEqual(collapsed, map[int]bool{1: true}) {
		t.Fatal("source mutated")
	}
}
