package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func TestSkinProseFontKeepsCodeAndMathTextMeasurementInSync(t *testing.T) {
	if err := theme.RegisterSkinFonts(); err != nil {
		t.Fatal(err)
	}
	st := Style{DocumentLayout: true, FontFamily: theme.ClaudeProseFamily, FontSize: 14, FG: ui.Hex("#262624")}
	spans := Spans("Claude prose `inline code`", st)
	code := false
	for _, span := range spans {
		if span.Text == "inline code" {
			code = true
			if span.Font != theme.RenderPreferences().Code.Family {
				t.Fatal("serif leaked into code", span)
			}
		}
	}
	if !code {
		t.Fatal("inline code missing")
	}
	atoms := appendTextAtoms(nil, ui.Span{Text: "Serif Āé", Size: 14}, st, "", 1000)
	if len(atoms) == 0 {
		t.Fatal("prose atoms missing")
	}
	var gotWidth float32
	for _, atom := range atoms {
		gotWidth += atom.width
	}
	wantWidth, _ := ui.ShapeTextLine("Serif Āé", ui.Font{Family: st.FontFamily, Size: 14}, 0).Size()
	if difference := gotWidth - wantWidth; difference < -0.02 || difference > 0.02 {
		t.Fatal("math-adjacent prose width uses a different font", gotWidth, wantWidth)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, "| Heading | Value |\n|---|---|\n| Āé serif | Code |", st)
	}, 540, 180)
	if _, ok := tt.Find("Āé serif"); !ok {
		t.Fatal("serif table cell missing", tt.Texts())
	}
}
