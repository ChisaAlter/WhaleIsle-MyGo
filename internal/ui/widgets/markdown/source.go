package markdown

import (
	"path/filepath"
	"strings"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// maxSourceHighlightBytes skips chroma for huge buffers (UI stays responsive).
const maxSourceHighlightBytes = 120_000

// LanguageFromPath maps a file path extension to a chroma lexer name.
// Empty string means no dedicated language (caller may still Analyse).
func LanguageFromPath(path string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if ext == "" {
		return ""
	}
	switch ext {
	case "md", "markdown":
		return "markdown"
	case "py", "pyw":
		return "python"
	case "js", "mjs", "cjs":
		return "javascript"
	case "ts":
		return "typescript"
	case "tsx":
		return "tsx"
	case "jsx":
		return "jsx"
	case "yml":
		return "yaml"
	case "rs":
		return "rust"
	case "cs":
		return "csharp"
	case "kt", "kts":
		return "kotlin"
	case "sh", "bash", "zsh":
		return "bash"
	case "ps1":
		return "powershell"
	case "h":
		return "c"
	case "hpp", "hh", "hxx", "cc", "cxx", "cpp":
		return "cpp"
	default:
		return ext
	}
}

// CanSourceHighlight reports whether path/code should use SourceView.
func CanSourceHighlight(path, code string) bool {
	if code == "" || len(code) >= maxSourceHighlightBytes {
		return false
	}
	lang := LanguageFromPath(path)
	switch lang {
	case "csv", "tsv":
		return false
	}
	return true
}

// SourceSpans highlights a whole source file using chroma (by path or analyser).
func SourceSpans(path, code string, st Style) []ui.Span {
	st.FontSize = theme.CurrentPreferences().Code.Size
	lang := LanguageFromPath(path)
	switch lang {
	case "markdown":
		return Spans(code, st)
	case "csv", "tsv":
		return []ui.Span{{Text: code, Size: st.FontSize, Color: st.FG, Font: theme.RenderPreferences().Code.Family}}
	}
	return highlightedCodeSpans(lang, code, st)
}

// SourceView paints highlighted source in RichText.
func SourceView(c *ui.Context, path, code string, st Style) {
	st.FontSize = theme.CurrentPreferences().Code.Size
	spans := SourceSpans(path, code, st)
	if len(spans) == 0 {
		ui.Text(c, code).Font(theme.RenderPreferences().Code.Family).FontSize(st.FontSize).TextColor(st.FG)
		return
	}
	ui.RichText(c, spans...).Font(theme.RenderPreferences().Code.Family).FontSize(st.FontSize).TextColor(st.FG)
}
