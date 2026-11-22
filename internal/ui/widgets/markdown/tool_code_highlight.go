package markdown

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/egoist/mygo/ui"
)

// toolCodeSpans shares the shiki-sheet coloring with fenced code blocks.
// Disable EOL normalization and the lexer's synthetic final newline so the
// token stream preserves authored tabs, CRLF and final blanks.
func toolCodeSpans(language, code string, st Style) []ui.Span {
	return codeSpans(language, code, st, &chroma.TokeniseOptions{State: "root", Nested: true, EnsureLF: false}, nil)
}
