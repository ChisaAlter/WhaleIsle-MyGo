package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func TestSpansBoldAndCode(t *testing.T) {
	st := Style{FontSize: 14, FG: ui.Color{R: 1, G: 1, B: 1, A: 1}}
	spans := Spans("Hello **world** and `code`", st)
	if len(spans) < 3 {
		t.Fatalf("spans %#v", spans)
	}
	joined := concat(spans)
	if !strings.Contains(joined, "Hello ") || !strings.Contains(joined, "world") || !strings.Contains(joined, "code") {
		t.Fatalf("text %q", joined)
	}
	var sawBold, sawCode bool
	codeFont := theme.CurrentPreferences().Code
	for _, s := range spans {
		if s.Text == "world" && s.Weight >= 600 {
			sawBold = true
		}
		if s.Text == "code" && s.Font == codeFont.Family && s.Size == codeFont.Size {
			sawCode = true
		}
	}
	if !sawBold || !sawCode {
		t.Fatalf("bold=%v code=%v spans=%#v", sawBold, sawCode, spans)
	}
}

func TestSpansHeading(t *testing.T) {
	st := Style{FontSize: 14, FG: ui.Color{R: 1, G: 1, B: 1, A: 1}}
	spans := Spans("# Title\n\nBody", st)
	if len(spans) == 0 {
		t.Fatal("empty")
	}
	if spans[0].Text != "Title" || spans[0].Weight < 600 {
		t.Fatalf("heading span %#v", spans[0])
	}
}

func TestSpansEmpty(t *testing.T) {
	if got := Spans("   ", Style{FontSize: 12}); got != nil {
		t.Fatalf("got %#v", got)
	}
}

func concat(spans []ui.Span) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}
