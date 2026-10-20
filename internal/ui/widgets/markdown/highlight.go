package markdown

import (
	"strings"
	"whaleisle/internal/ui/theme"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/egoist/mygo/ui"
)

// chromaPalette picks the Original CodeBlock's light or dark token sheet.
// The conversation publishes one surface per mode, so a light palette would
// leave unreadable tokens on a dark transcript.
func chromaPalette(dark bool) *chroma.Style {
	name := "github"
	if dark {
		name = "github-dark"
	}
	if style := styles.Get(name); style != nil {
		return style
	}
	if style := styles.Get("github"); style != nil {
		return style
	}
	return styles.Fallback
}

// highlightedCodeSpans tokenizes a fenced code block with chroma.
// Falls back to a single monospace span when the language is unknown.
func highlightedCodeSpans(lang, code string, st Style) []ui.Span {
	code = strings.TrimRight(code, "\n")
	if code == "" {
		return nil
	}
	size := CodeSize(st)
	font := theme.RenderPreferences().Code.Family
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		return withLineNumbers([]ui.Span{{Text: code + "\n", Size: size, Background: st.CodeBG, Font: font}}, st)
	}
	lexer = chroma.Coalesce(lexer)
	style := chromaPalette(st.Dark)
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return withLineNumbers([]ui.Span{{Text: code + "\n", Size: size, Background: st.CodeBG, Font: font}}, st)
	}
	var out []ui.Span
	for _, tok := range it.Tokens() {
		if tok.Value == "" {
			continue
		}
		out = append(out, ui.Span{
			Text:       tok.Value,
			Size:       size,
			Color:      chromaColor(style, tok.Type, st.FG),
			Background: st.CodeBG,
			Font:       font,
		})
	}
	if len(out) == 0 {
		return withLineNumbers([]ui.Span{{Text: code + "\n", Size: size, Background: st.CodeBG, Font: font}}, st)
	}
	out = append(out, ui.Span{Text: "\n", Size: size, Background: st.CodeBG, Font: font})
	return withLineNumbers(out, st)
}

// withLineNumbers prefixes each source line with a right-aligned counter in the
// code gutter. Numbers are their own muted spans, so the block still renders as
// one wrapped RichText and stays selectable/copyable as one source.
func withLineNumbers(spans []ui.Span, st Style) []ui.Span {
	if !st.LineNumbers {
		return spans
	}
	size := CodeSize(st)
	font := theme.RenderPreferences().Code.Family
	width := 2
	lines := 1
	for _, span := range spans {
		lines += strings.Count(span.Text, "\n")
	}
	for n := lines; n >= 10; n /= 10 {
		width++
	}
	out := make([]ui.Span, 0, len(spans)+lines)
	line := 1
	prefix := func() {
		number := padNumber(line, width)
		out = append(out, ui.Span{Text: number + "  ", Size: size, Color: st.Muted.Alpha(.75), Background: st.CodeBG, Font: font})
		line++
	}
	prefix()
	for _, span := range spans {
		parts := strings.Split(span.Text, "\n")
		for index, part := range parts {
			if index > 0 {
				out = append(out, ui.Span{Text: "\n", Size: size, Background: st.CodeBG, Font: font})
				prefix()
			}
			if part == "" {
				continue
			}
			copied := span
			copied.Text = part
			out = append(out, copied)
		}
	}
	return out
}

func padNumber(n, width int) string {
	text := ""
	for n > 0 {
		text = string(rune('0'+n%10)) + text
		n /= 10
	}
	if text == "" {
		text = "0"
	}
	for len(text) < width {
		text = " " + text
	}
	return text
}

func chromaColor(style *chroma.Style, t chroma.TokenType, fallback ui.Color) ui.Color {
	entry := style.Get(t)
	if !entry.IsZero() && entry.Colour.IsSet() {
		c := entry.Colour
		return ui.RGBA(c.Red(), c.Green(), c.Blue(), 255)
	}
	return fallback
}
