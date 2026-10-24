package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestToolCodeSourceCopyPreservesTabsAndSingleTerminatorRule(t *testing.T) {
	for _, code := range []string{"\tconst n = 1;\r\n\n\n", " \n", "\n", "const text = '<b>&</b>';"} {
		t.Run(code, func(t *testing.T) {
			st := Style{FontSize: 14, FG: ui.RGB(20, 20, 20), Muted: ui.RGB(100, 100, 100), CodeBG: ui.RGB(249, 250, 251)}
			want := strings.TrimSuffix(code, "\n")
			spans := toolCodeSpans("typescript", want, st)
			var joined strings.Builder
			for _, span := range spans {
				joined.WriteString(span.Text)
			}
			if joined.String() != want {
				t.Fatalf("display altered source %q want%q", joined.String(), want)
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { ToolCodeBlock(c, "typescript", code, st, 0) })
			}, 400, 500)
			tt.SetTime(time.Unix(12000, 0))
			if err := tt.Click("Copy code"); err != nil {
				t.Fatal(err)
			}
			if tt.Clipboard() != want {
				t.Fatalf("copy=%q want%q", tt.Clipboard(), want)
			}
		})
	}
}

func TestToolCodeWrapChangesSourceGeometryAndRetainsChoice(t *testing.T) {
	st := Style{FontSize: 14, FG: ui.RGB(20, 20, 20), Muted: ui.RGB(100, 100, 100), CodeBG: ui.RGB(249, 250, 251)}
	code := "const sentence = '" + strings.Repeat("wide ", 25) + "';"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { ToolCodeBlock(c, "typescript", code, st, 0).Label("card") })
	}, 320, 500)
	tt.SetTime(time.Unix(12000, 0))
	wrapped, _ := tt.Find("card")
	if err := tt.Click("Wrap lines"); err != nil {
		t.Fatal(err)
	}
	unwrapped, _ := tt.Find("card")
	if wrapped.H <= unwrapped.H {
		t.Fatalf("wrap did not change layout %v ->%v", wrapped, unwrapped)
	}
	tt.Frame()
	still, _ := tt.Find("card")
	if still.H != unwrapped.H {
		t.Fatal("wrap choice reset")
	}
	tt.Key(0, ui.KeyEnter)
	rewrapped, _ := tt.Find("card")
	if rewrapped.H != wrapped.H {
		t.Fatal("focused Enter failed wrap")
	}
}

func TestToolCodeHighlightUsesInstalledLightAndDarkTokens(t *testing.T) {
	for _, tc := range []struct {
		dark                         bool
		keyword, stringColor, number string
	}{{false, "#d6336c", "#2f9e44", "#1c7ed6"}, {true, "#faa2c1", "#69db7c", "#4dabf7"}} {
		spans := toolCodeSpans("typescript", "const label = 'text'; const n = 42;", Style{FG: ui.RGB(20, 20, 20), Dark: tc.dark})
		foundKeyword, foundString, foundNumber := false, false, false
		for _, span := range spans {
			if strings.Contains(span.Text, "const") {
				foundKeyword = span.Color == ui.Hex(tc.keyword)
			}
			if strings.Contains(span.Text, "text") {
				foundString = span.Color == ui.Hex(tc.stringColor)
			}
			if strings.Contains(span.Text, "42") {
				foundNumber = span.Color == ui.Hex(tc.number)
			}
		}
		if !foundKeyword || !foundString || !foundNumber {
			t.Fatalf("dark%v palette missing: keyword%v string%v number%v", tc.dark, foundKeyword, foundString, foundNumber)
		}
	}
}
