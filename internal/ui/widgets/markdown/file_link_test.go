package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestLocalLinkActivationDefersUntilSettlement(t *testing.T) {
	streaming := true
	opened := ""
	source := "See [Report](./report.md#L12)."
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, source, Style{DocumentLayout: true, Streaming: streaming, OnOpenFile: func(path string) { opened = path }})
		})
	}, 400, 200)
	paragraph, ok := tt.Find("See Report.")
	if !ok {
		t.Fatalf("streaming text missing: %q", tt.Texts())
	}
	tt.ClickAt(paragraph.X+40, paragraph.Y+paragraph.H/2)
	if opened != "" {
		t.Fatal("streaming file link activated")
	}
	streaming = false
	tt.Frame()
	if err := tt.Click("Report"); err != nil {
		t.Fatal(err)
	}
	if opened != "./report.md#L12" {
		t.Fatalf("settled local link=%q", opened)
	}
}

func TestLinkedLocalImageKeepsAnchorDestination(t *testing.T) {
	opened := ""
	st := Style{DocumentLayout: true, OnOpenFile: func(path string) { opened = path }, RenderImage: func(c *ui.Context, _, _ string) bool {
		ui.Box(c).Width(100).Height(40).Label("Raster image")
		return true
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, "[![](./image.png)](./report.md#L2)", st) })
	}, 400, 200)
	if err := tt.Click("./report.md#L2"); err != nil {
		t.Fatal(err)
	}
	if opened != "./report.md#L2" {
		t.Fatalf("image anchor opened=%q", opened)
	}
}

func TestDeliverableDocumentLinkStylingAndActivation(t *testing.T) {
	source := "[下载 Word 文档](docs/PDA物料盘点优化前后对比及优化内容_2026-10-30.docx)"
	spans := Spans(source, Style{Dark: true})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Underline {
		t.Fatal("link span should not have permanent underline at rest")
	}
	if spans[0].Color != ui.Hex("#6ea8ff") {
		t.Fatalf("unexpected dark link color: %+v", spans[0].Color)
	}
	opened := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			View(c, source, Style{
				Dark:       true,
				OnOpenFile: func(path string) { opened = path },
			})
		})
	}, 400, 120)
	if err := tt.Click("下载 Word 文档"); err != nil {
		t.Fatal(err)
	}
	if opened != "docs/PDA物料盘点优化前后对比及优化内容_2026-10-30.docx" {
		t.Fatalf("deliverable link opened=%q", opened)
	}
}
