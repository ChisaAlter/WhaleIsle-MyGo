package markdown

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/egoist/mygo/ui"
)

// toolCodeSpans uses the installed shiki CSS-variable palette with the native
// TypeScript lexer. Chroma and shiki may classify individual tokens differently.
// Disable EOL normalization and the lexer's synthetic final newline so the
// token stream preserves authored tabs, CRLF and final blanks.
func toolCodeSpans(language, code string, st Style) []ui.Span {
	return codeSpans(language, code, st, &chroma.TokeniseOptions{State: "root", Nested: true, EnsureLF: false}, func(t chroma.TokenType) ui.Color {
		light, dark := "", ""
		switch {
		case t.InCategory(chroma.Comment):
			light, dark = "#868e96", "#adb5bd"
		case t.InCategory(chroma.Keyword):
			light, dark = "#d6336c", "#faa2c1"
		case t == chroma.LiteralStringInterpol:
			light, dark = "#2b8a3e", "#8ce99a"
		case t.InSubCategory(chroma.LiteralString):
			light, dark = "#2f9e44", "#69db7c"
		case t.InSubCategory(chroma.LiteralNumber) || t == chroma.NameConstant || t.InSubCategory(chroma.NameBuiltin):
			light, dark = "#1c7ed6", "#4dabf7"
		case t.InSubCategory(chroma.NameFunction):
			light, dark = "#6741d9", "#b197fc"
		case t == chroma.Punctuation || t.InCategory(chroma.Operator):
			light, dark = "#495057", "#ced4da"
		default:
			return st.FG
		}
		if st.Dark {
			return ui.Hex(dark)
		}
		return ui.Hex(light)
	})
}
