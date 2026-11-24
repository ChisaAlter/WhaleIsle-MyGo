package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// transcriptMarkdownStyle is the single markdown preset for conversation
// surfaces: document block ladder, localized code toolbar and the Original
// blockquote inset. Keeping it in one place stops a second surface from
// drifting back to the flat span run.
func transcriptMarkdownStyle(fg, muted, field, accent, hover ui.Color, size float32, dark bool, secondary ui.Color) markdown.Style {
	wrapLabel, noWrapLabel := i18n.T(i18n.KeyMarkdownWrap), i18n.T(i18n.KeyMarkdownNoWrap)
	tokens := design.Base(designScheme(dark))
	return markdown.Style{
		DocumentLayout: true,
		FontFamily:     theme.FontForSkin(theme.RenderPreferences(), theme.SkinForMode(theme.CurrentPreferences(), dark), theme.SkinProseFont).Family,
		FontSize:       size,
		FG:             fg,
		Muted:          muted,
		Marker:         secondary,
		CodeBG:         tokens.Code,
		Accent:         accent,
		CodeSurface: func(e ui.Element, radius float32) ui.Element {
			return e.Background(tokens.Code).Border(design.BorderWidth, tokens.Border).Radius(radius)
		},
		InlineCodeBG:  tokens.CodeHighlight,
		Dark:          dark,
		QuoteInset:    14,
		ToolCodeHover: hover,
		CopyLabel:     i18n.T(i18n.KeyMarkdownCopyCode),
		CopiedLabel:   i18n.T(i18n.KeyMarkdownCopied),
		WrapLabel:     wrapLabel,
		NoWrapLabel:   noWrapLabel,
		CodeLabel:     i18n.T(i18n.KeyMarkdownCode),
	}
}

func transcriptMarkdownStyleFor(c *ui.Context, size float32) markdown.Style {
	t := design.For(c)
	st := transcriptMarkdownStyle(t.Foreground, t.MutedForeground, t.Code, t.Primary, t.Accent, size, t.IsDark(), t.MutedForeground)
	st.CodeBG = t.Code
	st.CodeSurface = func(e ui.Element, radius float32) ui.Element {
		return e.Background(t.Code).Border(design.BorderWidth, t.Border).Radius(radius)
	}
	st.InlineCodeBG = t.CodeHighlight
	st.TableHeader = t.Muted
	st.QuoteBorder = t.Border
	st.RuleBorder = t.Border
	return st
}

func designScheme(dark bool) design.Scheme {
	if dark {
		return design.Dark
	}
	return design.Light
}

// darkSurface reports whether a color pair reads as a dark theme, judged from
// the primary foreground (the theme package exposes no mode on a color).
func darkSurface(fg ui.Color) bool {
	luma := 0.2126*float32(fg.R) + 0.7152*float32(fg.G) + 0.0722*float32(fg.B)
	return luma > 0x88
}
