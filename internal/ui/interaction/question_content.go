package interaction

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// Keep prose within the current question's available width, without line caps
// or summary truncation. Header and question remain separate plain-text fields.
func questionText(c *ui.Context, text string, size float32, color ui.Color) {
	ui.Text(c, text).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(size)).TextColor(color)
}

func questionDetail(c *ui.Context, detail string, fg, muted ui.Color) {
	if detail == "" {
		return
	}
	// Use the same safe Markdown renderer as chat, including exact-source code
	// copy, horizontal scrolling and the per-code-block line-wrap toggle.
	ui.Column(c.Key("question-detail")).FillWidth().MinWidth(0).Children(func() {
		markdown.View(c, detail, markdown.Style{
			FontSize:        theme.InterfaceSize(12),
			FG:              fg,
			Muted:           muted,
			CodeBG:          muted.Alpha(0.08),
			Accent:          fg,
			CodeWrapDefault: true,
			QuoteInset:      14,
			CopyLabel:       i18n.T(i18n.KeyMarkdownCopyCode),
			CopiedLabel:     i18n.T(i18n.KeyMarkdownCopied),
			WrapLabel:       i18n.T(i18n.KeyMarkdownWrap),
			NoWrapLabel:     i18n.T(i18n.KeyMarkdownNoWrap),
			CodeLabel:       i18n.T(i18n.KeyMarkdownCode),
		})
	})
}
