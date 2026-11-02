package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestStreamingCodeHighlightBoundsLexerWorkAndSettlesMultilineTokens(t *testing.T) {
	parsed := 0
	state := codeHighlightCache{tokenize: func(language, code string, st Style) []ui.Span {
		parsed += len(code)
		return highlightedCodeSpans(language, code, st)
	}}
	code := "package main\n/* ongoing comment\n"
	st := Style{Streaming: true, FontSize: 14}
	for index := range 400 {
		code += fmt.Sprintf("comment line %d\n", index)
		if got := concat(state.update("go", code, st)); got != strings.TrimRight(code, "\n") {
			t.Fatalf("growing code disappeared: %q", got)
		}
	}
	if parsed > len(code)*4 {
		t.Fatalf("lexer work grew with every full prefix: parsed=%d source=%d", parsed, len(code))
	}
	t.Logf("400 appended code lines: source=%d lexer-input=%d", len(code), parsed)
	code += "*/\nvar answer = 42\n"
	st.Streaming = false
	got := state.update("go", code, st)
	want := highlightedCodeSpans("go", strings.TrimRight(code, "\n"), st)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatal("settlement did not fully recolor multiline source")
	}
	before := parsed
	state.update("go", code, st)
	if before != parsed {
		t.Fatal("idle frame reran lexer")
	}
}
