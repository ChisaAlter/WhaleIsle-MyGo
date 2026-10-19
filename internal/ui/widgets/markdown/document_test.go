package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func documentText(t *testing.T, tt *ui.Tester, label string) string {
	t.Helper()
	for _, text := range tt.Texts() {
		if strings.TrimSpace(text) == strings.TrimSpace(label) {
			return text
		}
	}
	t.Fatalf("document text %q not found in %q", label, tt.Texts())
	return ""
}

func TestDocumentInlineURLsKeepTextFlowAndRequireActivation(t *testing.T) {
	var opened []string
	st := Style{OnOpenURL: func(href string) { opened = append(opened, href) }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, "Before [Docs](https://example.invalid/docs) after. [Relative](guide.md) [Unsafe](javascript:alert(1)) [Mail](mailto:fixture@example.invalid)", st)
		})
	}, 720, 300)
	if len(opened) != 0 {
		t.Fatal("rendering navigated")
	}
	before, beforeOK := tt.Find("Before Docs after. Relative Unsafe Mail")
	docs, docsOK := tt.Find("Docs")
	if !beforeOK || !docsOK || docs.Y != before.Y || docs.X <= before.X {
		t.Fatalf("link left its paragraph: before=%v docs=%v", before, docs)
	}
	font := theme.CurrentPreferences().Chat
	for _, prefix := range []string{"Before Docs after. ", "Before Docs after. Relative "} {
		width, _ := ui.ShapeRichText([]ui.Span{{Text: prefix}}, ui.Font{Family: font.Family, Size: font.Size}, 0).Size()
		tt.ClickAt(before.X+width+2, before.Y+before.H/2)
	}
	if len(opened) != 0 {
		t.Fatal("relative or executable link became actionable")
	}
	if err := tt.Click("Docs"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://example.invalid/docs" {
		t.Fatalf("inline navigation lost its destination: %v", opened)
	}
	if err := tt.Click("Mail"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 2 || opened[1] != "mailto:fixture@example.invalid" {
		t.Fatalf("mailto destination changed: %v", opened)
	}
}

func TestDocumentLiteralHTMLAndFormattedLinkContents(t *testing.T) {
	source := "Before <b>literal</b> after.\n\n<section>\nRaw block\n</section>\n\n[**Bold** guide](https://example.invalid)"
	st := Style{LiteralHTML: true, OnOpenURL: func(string) {}}
	spans := Spans(source, st)
	var body strings.Builder
	bold := false
	for _, span := range spans {
		body.WriteString(span.Text)
		if span.Text == "Bold" && span.Weight == 700 {
			bold = true
		}
	}
	if !strings.Contains(body.String(), "<b>literal</b>") || !strings.Contains(body.String(), "<section>\nRaw block\n</section>") || !bold {
		t.Fatalf("literal markup or link content formatting lost: %q %+v", body.String(), spans)
	}
}

func TestDocumentImagesStayInOrderIncludingTableCells(t *testing.T) {
	var rendered []string
	st := Style{RenderImage: func(c *ui.Context, src, alt string) bool {
		rendered = append(rendered, src)
		if !strings.HasPrefix(src, "https://") {
			return false
		}
		ui.Text(c, "Image "+alt)
		return true
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, "First\n\n![wide](https://example.invalid/wide.png)\n\nMiddle\n\n| Item |\n| --- |\n| ![cell](https://example.invalid/cell.png) |\n\n![Relative alt](./relative.png)\n\nLast", st)
		})
	}, 700, 700)
	if len(rendered) != 3 || rendered[0] != "https://example.invalid/wide.png" || rendered[1] != "https://example.invalid/cell.png" || rendered[2] != "./relative.png" {
		t.Fatalf("images escaped source order or table ownership: %v", rendered)
	}
	lastY := float32(-1)
	for _, label := range []string{"First", "Image wide", "Middle", "Image cell", "Relative alt", "Last"} {
		r, ok := tt.Find(documentText(t, tt, label))
		if !ok || r.Y < lastY {
			t.Fatalf("document order lost at %q: %v (previous y=%v)", label, r, lastY)
		}
		lastY = r.Y
	}
}

func TestDocumentURLTargetsKeepMathSpanOffsets(t *testing.T) {
	st := Style{FontSize: 14, urlTargets: map[int]string{8: "https://example.invalid/exact"}}
	spans := []ui.Span{{Text: "Before ", Size: 14}, {Text: "Linked words", Size: 14}}
	atoms := mathAtoms(spans, st, nil, 7, nil, 80)
	linked := 0
	for _, atom := range atoms {
		if atom.url != "" {
			linked++
			if atom.url != "https://example.invalid/exact" || atom.target != "" {
				t.Fatalf("math inline URL became a file target: %+v", atom)
			}
		}
	}
	if linked == 0 {
		t.Fatal("math layout discarded inline URL ownership")
	}
}

func TestDocumentLinkedImageKeepsItsOriginalInlineDestination(t *testing.T) {
	for _, available := range []bool{true, false} {
		t.Run(map[bool]string{true: "rendered", false: "alt"}[available], func(t *testing.T) {
			var opened []string
			st := Style{OnOpenURL: func(href string) { opened = append(opened, href) }, RenderImage: func(c *ui.Context, _, _ string) bool {
				if available {
					ui.Box(c).Width(120).Height(60).Label("Rendered diagram")
				}
				return available
			}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { View(c, "[![Diagram](https://example.invalid/image.png)](https://example.invalid/guide)", st) })
			}, 360, 240)
			if len(opened) != 0 {
				t.Fatal("linked image navigated while rendering")
			}
			if err := tt.Click("Diagram"); err != nil {
				t.Fatal(err)
			}
			if len(opened) != 1 || opened[0] != "https://example.invalid/guide" {
				t.Fatalf("image lost or duplicated enclosing link action: %v", opened)
			}
		})
	}
}

func TestDocumentLayoutResetsOuterMarginsAndCollapsesInteriorMargins(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(320).Label("Document container").Children(func() {
			View(c, "# Heading\n\nParagraph", Style{DocumentLayout: true})
		})
	}, 400, 240)
	heading, _ := tt.Find(documentText(t, tt, "Heading"))
	paragraph, _ := tt.Find(documentText(t, tt, "Paragraph"))
	container, _ := tt.Find("Document container")
	if heading.Y != container.Y || paragraph.Y-heading.Y-heading.H != 16 || paragraph.Y+paragraph.H != container.Y+container.H {
		t.Fatalf("source first/last-child reset or collapsed margin lost: container=%v heading=%v paragraph=%v", container, heading, paragraph)
	}
}

func TestDocumentFormulaURLsSurviveRenderedAndPendingMath(t *testing.T) {
	formula := mathFormula{TeX: "x", Raw: "$x$"}
	svg, err := ui.ParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, ready := range []bool{false, true} {
		st := Style{FontSize: 14, maths: map[int]mathFormula{3: formula}, urlTargets: map[int]string{3: "https://example.invalid/formula"}}
		images := map[mathFormula]mathImage{}
		if ready {
			images[formula] = mathImage{ready: true, svg: svg, width: 1, height: 1}
		}
		atoms := mathAtoms([]ui.Span{{Text: formula.Raw, Size: 14}}, st, nil, 3, images, 200)
		if len(atoms) == 0 {
			t.Fatal("math disappeared")
		}
		for _, atom := range atoms {
			if atom.url != "https://example.invalid/formula" || atom.target != "" {
				t.Fatalf("formula link lost in ready=%v: %+v", ready, atom)
			}
		}
	}
}

func TestDocumentEmptyAltImageCannotLoseOrBorrowEnclosingLink(t *testing.T) {
	var opened []string
	st := Style{OnOpenURL: func(href string) { opened = append(opened, href) }, RenderImage: func(c *ui.Context, _, _ string) bool {
		ui.Box(c).Width(100).Height(40).Label("Raster")
		return true
	}}
	var source = "[![](https://example.invalid/image.png)](https://example.invalid/guide)"
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, source, st) }) }, 360, 280)
	if err := tt.Click("https://example.invalid/guide"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://example.invalid/guide" {
		t.Fatalf("empty-alt image lost its enclosing link: %v", opened)
	}
	opened = nil
	source = "![](https://example.invalid/image.png) [Docs](https://example.invalid/other)"
	tt.Frame()
	if err := tt.Click("Raster"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 0 {
		t.Fatalf("unlinked empty-alt image borrowed the next link: %v", opened)
	}
	if err := tt.Click("Docs"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://example.invalid/other" {
		t.Fatalf("next text link changed identity: %v", opened)
	}
}

func TestDocumentRepeatedImageURLsRetainSeparateOccurrences(t *testing.T) {
	st := Style{RenderImage: func(c *ui.Context, source, alt string) bool {
		root := ui.Box(c.Key(source)).Width(100).Height(40)
		owner := ui.Local(root, "owner", func() string { return alt })
		root.Children(func() { ui.Text(c, "Retained "+*owner) })
		return true
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, "![first](https://example.invalid/repeat.png) ![second](https://example.invalid/repeat.png)", st)
		})
	}, 360, 280)
	for range 2 {
		tt.Frame()
		first, firstOK := tt.Find("Retained first")
		second, secondOK := tt.Find("Retained second")
		if !firstOK || !secondOK || second.Y <= first.Y {
			t.Fatalf("same-URL occurrences shared retained state: %v %v", first, second)
		}
	}
}
