package ansi

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func plain(lines [][]ui.Span) string {
	parts := []string{}
	for _, line := range lines {
		var text strings.Builder
		for _, span := range line {
			text.WriteString(span.Text)
		}
		parts = append(parts, text.String())
	}
	return strings.Join(parts, "\n")
}

func TestReplayControlsWideColumnsAndTerminator(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"100%\rOK", "OK0%"}, {"abc\bX", "abX"}, {"abc\r\x1b[Kxy", "xy"}, {"界x\rA", "A x"},
		{"e\u0301x\rA", "Ax"}, {"a\tb", "a       b"}, {"\x1b]8;;https://inert.invalid\x07text\x1b]8;;\x1b\\", "text"},
		{"line\n\x1b[0m", "line"}, {"line\n\n", "line\n"}, {"\x1b[0m\x1b]0;title\x07", ""},
	} {
		if got := plain(Parse(tc.raw, Palette{})); got != tc.want {
			t.Fatalf("%q→%q want %q", tc.raw, got, tc.want)
		}
	}
}

func TestForegroundTokensAndExplicitBackgroundLiteralPair(t *testing.T) {
	p := Palette{Primary: ui.RGB(1, 2, 3), Error: ui.RGB(4, 5, 6), ErrorSecondary: ui.RGB(7, 8, 9), Background: ui.RGB(10, 11, 12), DimBase: ui.RGB(13, 14, 15)}
	line := Parse("\x1b[31mred\x1b[91mbright\x1b[39mplain\x1b[31;47mpair\x1b[0m", p)[0]
	if len(line) != 4 || line[0].Color != p.Error || line[1].Color != p.ErrorSecondary || line[2].Color.A != 0 || line[3].Color != ui.RGB(187, 0, 0) || line[3].Background != ui.RGB(187, 187, 187) {
		t.Fatalf("colors: %+v", line)
	}
	line = Parse("\x1b[1;3;4;9mboth\x1b[29munderline\x1b[22;23;24mplain", p)[0]
	if len(line) != 3 || line[0].Weight != 700 || !line[0].Italic || !line[0].Strikethrough || line[0].Underline || !line[1].Underline || line[2].Weight != 0 || line[2].Italic || line[2].Underline {
		t.Fatalf("decorations: %+v", line)
	}
	line = Parse("\x1b[38;2;10;20;30mtruecolor\x1b[39;2mdim\x1b[22;8mhidden", p)[0]
	if len(line) != 3 || line[0].Color != ui.RGB(10, 20, 30) || line[1].Color.A != 178 || line[2].Color != p.Background {
		t.Fatalf("color/dim/hidden: %+v", line)
	}
}
