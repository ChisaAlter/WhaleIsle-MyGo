package markdown

import (
	"strings"
	"whaleisle/internal/ui/theme"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/egoist/mygo/ui"
)

// highlightedCodeSpans tokenizes a fenced code block with chroma.
// Falls back to a single monospace span when the language is unknown.
func highlightedCodeSpans(lang, code string, st Style) []ui.Span {
	code = strings.TrimRight(code, "\n")
	if code == "" {
		return nil
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		return []ui.Span{codeSpan(code+"\n", st)}
	}
	lexer = chroma.Coalesce(lexer)
	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return []ui.Span{codeSpan(code+"\n", st)}
	}
	var out []ui.Span
	for _, tok := range it.Tokens() {
		if tok.Value == "" {
			continue
		}
		out = append(out, ui.Span{
			Text:       tok.Value,
			Size:       theme.CurrentPreferences().Code.Size,
			Color:      chromaColor(style, tok.Type, st.FG),
			Background: st.CodeBG,
			Font:       theme.CurrentPreferences().Code.Family,
		})
	}
	if len(out) == 0 {
		return []ui.Span{codeSpan(code+"\n", st)}
	}
	out = append(out, ui.Span{Text: "\n", Size: theme.CurrentPreferences().Code.Size, Background: st.CodeBG, Font: theme.CurrentPreferences().Code.Family})
	return out
}

func chromaColor(style *chroma.Style, t chroma.TokenType, fallback ui.Color) ui.Color {
	entry := style.Get(t)
	if !entry.IsZero() && entry.Colour.IsSet() {
		c := entry.Colour
		return ui.RGBA(c.Red(), c.Green(), c.Blue(), 255)
	}
	return fallback
}
