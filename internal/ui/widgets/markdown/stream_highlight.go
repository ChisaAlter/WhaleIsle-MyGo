package markdown

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// Chroma's public iterator cannot resume an arbitrary lexer stack. While code
// streams, growing checkpoints re-tokenize the accumulated code geometrically;
// the appended suffix remains readable monospace until the next checkpoint.
// Total lexer input is linear in final code size. Settlement always highlights
// the complete source, including multiline strings and comments.
type codeHighlightCache struct {
	language    string
	code        string
	checkpoint  string
	spans       []ui.Span
	style       Style
	streaming   bool
	initialized bool
	font        string
	tokenize    func(string, string, Style) []ui.Span
}

func (s *codeHighlightCache) update(language, code string, st Style) []ui.Span {
	code = strings.TrimRight(code, "\n")
	sameStyle := s.style.Dark == st.Dark && s.style.FontSize == st.FontSize && s.style.FG == st.FG && s.style.Muted == st.Muted && s.style.CodeBG == st.CodeBG && s.style.LineNumbers == st.LineNumbers
	reset := !s.initialized || s.language != language || !sameStyle || s.font != theme.RenderPreferences().Code.Family || !strings.HasPrefix(code, s.code)
	if reset {
		s.checkpoint = ""
		s.spans = nil
	}
	if reset || (!st.Streaming && (s.streaming || s.code != code)) || (st.Streaming && len(code) >= max(64, len(s.checkpoint)*2)) {
		tokenize := s.tokenize
		if tokenize == nil {
			tokenize = highlightedCodeSpans
		}
		s.spans = tokenize(language, code, st)
		s.checkpoint = code
	}
	s.language, s.code, s.style, s.streaming, s.initialized = language, code, st, st.Streaming, true
	s.font = theme.RenderPreferences().Code.Family
	if len(s.checkpoint) == len(code) {
		return s.spans
	}
	out := append([]ui.Span(nil), s.spans...)
	return append(out, ui.Span{Text: code[len(s.checkpoint):], Size: CodeSize(st), Color: st.FG, Background: st.CodeBG, Font: theme.RenderPreferences().Code.Family})
}
