package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

// transcriptMarkdownStyle is the single markdown preset for conversation
// surfaces: document block ladder, localized code toolbar and the Original
// blockquote inset. Keeping it in one place stops a second surface from
// drifting back to the flat span run.
func transcriptMarkdownStyle(fg, muted, field, accent ui.Color, size float32, dark bool) markdown.Style {
	wrapLabel, noWrapLabel := i18n.T(i18n.KeyMarkdownWrap), i18n.T(i18n.KeyMarkdownNoWrap)
	return markdown.Style{
		DocumentLayout: true,
		FontSize:       size,
		FG:             fg,
		Muted:          muted,
		CodeBG:         field,
		Accent:         accent,
		Dark:           dark,
		QuoteInset:     14,
		CopyLabel:      i18n.T(i18n.KeyMarkdownCopyCode),
		CopiedLabel:    i18n.T(i18n.KeyMarkdownCopied),
		WrapLabel:      wrapLabel,
		NoWrapLabel:    noWrapLabel,
		CodeLabel:      i18n.T(i18n.KeyMarkdownCode),
	}
}

// darkSurface reports whether a color pair reads as a dark theme, judged from
// the primary foreground (the theme package exposes no mode on a color).
func darkSurface(fg ui.Color) bool {
	luma := 0.2126*float32(fg.R) + 0.7152*float32(fg.G) + 0.0722*float32(fg.B)
	return luma > 0x88
}
