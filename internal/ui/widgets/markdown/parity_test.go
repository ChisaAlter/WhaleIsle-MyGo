package markdown

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFileLinkLocationsAndUnsafeDestinations(t *testing.T) {
	for _, tt := range []struct {
		value, path string
		line        int
	}{
		{"src/main.go#L24", "src/main.go", 24},
		{"src/main.go#L24-L30", "src/main.go", 24},
		{"C:/My%20Project/app.go:12", "C:/My Project/app.go", 12},
		{"./notes%23draft.md", "./notes#draft.md", 0},
	} {
		path, line, ok := ParseFileLink(tt.value)
		if !ok || path != tt.path || line != tt.line {
			t.Fatalf("parse %q=%q:%d %v", tt.value, path, line, ok)
		}
	}
	for _, value := range []string{"#L12", "report.md#L0", "report.md#L30-L24", "report.md#L24-anything", "report.md?download=1", "%xx.md", "javascript:alert(1)", "file:///tmp/a", "https://example.invalid/a", "//server/file", `\\server\file`, "notes%00.md", "notes%0a.md"} {
		if path, _, ok := ParseFileLink(value); ok {
			t.Fatalf("unsafe destination %q became %q", value, path)
		}
	}
}

func TestDocumentLocalLinksOpenFromTextAndTables(t *testing.T) {
	var opened []string
	st := Style{DocumentLayout: true, OnOpenFile: func(path string) { opened = append(opened, path) }}
	source := "Before [**Report**](<C:/My Project/report.md#L24-L30>) after.\n\n| File |\n| --- |\n| [Table file](src/table.go:12) |\n\n[Unsafe](javascript:alert(1))\n\n[Bad range](bad.go#L30-L24)"
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, source, st) }) }, 720, 450)
	if len(opened) != 0 {
		t.Fatal("render opened a file")
	}
	for _, label := range []string{"Report", "Table file"} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(opened, []string{"C:/My Project/report.md#L24", "src/table.go#L12"}) {
		t.Fatalf("local link locations=%q", opened)
	}
	for _, label := range []string{"Unsafe", "Bad range"} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if len(opened) != 2 {
		t.Fatalf("invalid link activated: %q", opened)
	}
}

func TestFootnotesRenderInFirstReferenceOrderWithRepeatedMarkers(t *testing.T) {
	source := "One[^second] then two[^first] and repeat[^second]. Missing[^missing].\n\n[^first]: First body with **bold**.\n\n[^second]: Second body.\n\n    Extra paragraph.\n\n[^unused]: Invisible body."
	spans := Spans(source, Style{})
	plain := concat(spans)
	if !strings.HasPrefix(plain, "One¹ then two² and repeat¹. Missing[^missing].") || !strings.Contains(plain, "1. Second body.") || !strings.Contains(plain, "2. First body with bold.") || strings.Contains(plain, "Invisible body") {
		t.Fatalf("footnote text=%q", plain)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, source, Style{DocumentLayout: true}) })
	}, 700, 600)
	second, _ := tt.Find(documentText(t, tt, "Second body."))
	first, _ := tt.Find(documentText(t, tt, "First body with bold. ↩"))
	if second.H <= 0 || first.Y <= second.Y {
		t.Fatalf("footnotes did not follow references: %v %v texts=%q", second, first, tt.Texts())
	}
}

func TestMathFenceSettlesThroughExistingFormulaRenderer(t *testing.T) {
	source := "```math\n\\frac{a}{b}\n```"
	formulas := map[int]mathFormula{}
	Spans(source, Style{maths: formulas})
	if len(formulas) != 1 {
		t.Fatalf("math fence remained code: %#v", formulas)
	}
	for _, formula := range formulas {
		if strings.TrimSpace(formula.TeX) != `\frac{a}{b}` || !formula.Display {
			t.Fatal(formula)
		}
	}
	formulas = map[int]mathFormula{}
	Spans(source, Style{Streaming: true, maths: formulas})
	if len(formulas) != 0 {
		t.Fatal("streaming math invoked formula renderer")
	}
}

func TestDocumentMultipleImagesAndPlaceholdersKeepSourceOrder(t *testing.T) {
	var rendered []string
	st := Style{DocumentLayout: true, RenderImage: func(c *ui.Context, path, alt string) bool {
		rendered = append(rendered, path)
		ui.Text(c, "Image "+path)
		return true
	}}
	source := "Before\n\n![one](./a_(1).png) ![two](<./My pics/b (2).png>)\n\nMiddle\n[image] attachment:legacy\nAfter"
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, source, st) }) }, 500, 600)
	want := []string{"./a_(1).png", "./My pics/b (2).png", "attachment:legacy"}
	if !reflect.DeepEqual(rendered, want) {
		t.Fatalf("image occurrences=%q", rendered)
	}
	last := float32(-1)
	for _, label := range []string{"Before", "Image ./a_(1).png", "Image ./My pics/b (2).png", "Middle", "Image attachment:legacy", "After"} {
		r, ok := tt.Find(label)
		if !ok || r.Y < last {
			t.Fatalf("order at %q: %v texts=%q", label, r, tt.Texts())
		}
		last = r.Y
	}
}
