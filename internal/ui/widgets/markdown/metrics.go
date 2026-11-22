package markdown

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// Metric ladder from the Original markdown sheet (gradient-shadow-text.css):
// body 14/24, inline code 12/19, fenced code 11/19, headings 21/30 · 19/28 ·
// 18/26. Everything scales from the body size, so the Settings font preference
// moves the whole document instead of one tier.
const (
	bodyLeading  = 10
	codeLeading  = 8
	inlineDelta  = 2
	blockDelta   = 3
	minCodeSize  = 11
	codeLineHigh = 19
)

// BodyLineHeight is the base 24px leading at the 14px default body size; the
// .compact variant runs the secondary 20px leading at its 13px size.
func BodyLineHeight(st Style) float32 {
	if st.Compact {
		if st.FontSize <= 0 {
			return 20
		}
		return st.FontSize + 7
	}
	if st.FontSize <= 0 {
		return 24
	}
	return st.FontSize + bodyLeading
}

// CodeSize is the fenced-code size: 11px at the 14px body default.
func CodeSize(st Style) float32 {
	size := st.FontSize - blockDelta
	if st.FontSize <= 0 {
		return minCodeSize
	}
	if size < minCodeSize {
		return minCodeSize
	}
	return size
}

// InlineCodeSize is the inline-code size: one step under the body, 12px at the
// 14px default. The .compact variant keeps inline code at the body size.
func InlineCodeSize(st Style) float32 {
	if st.Compact {
		return st.FontSize
	}
	size := st.FontSize - inlineDelta
	if st.FontSize <= 0 || size < minCodeSize {
		return minCodeSize
	}
	return size
}

// codeLineHeight keeps the Original's 19px code leading from the 11px size.
func codeLineHeight(st Style) float32 {
	if height := CodeSize(st) + codeLeading; height > codeLineHigh {
		return height
	}
	return codeLineHigh
}

// codeBodyFill is a fill that matches the code-card body: transparent when the
// card frosts (CodeSurface set — the frosted root shows between glyphs and rows
// instead of a second fill), else the CodeBG it always painted. Over a plain
// CodeBG root the transparent result is pixel-identical.
func codeBodyFill(st Style) ui.Color {
	if st.CodeSurface != nil {
		return ui.Transparent
	}
	return st.CodeBG
}

// codeBanner resolves the code-card header fill. The Original banner is
// --dsw-alias-markdown-code-block-banner, a muted tint over the body surface.
// codeBanner keeps an explicit CodeBanner fill when a caller sets one; the
// Original's default is a card of one surface — --dsw-alias-markdown-code-block
// and -banner share a token in the standard family — so the banner row paints
// nothing of its own.
func codeBanner(st Style) ui.Color {
	return st.CodeBanner
}

// InlineCodeBase resolves the inline-code chip fill: the skin's
// --dsw-alias-markdown-inline-code token, or the design-platform value the
// standard family ships (a translucent wash, so it sits right on any backdrop
// even without a frost wrapper). Exported for callers that pass the result
// through chrome.FrostInset when their token would be opaque.
func InlineCodeBase(dark bool) ui.Color {
	fallback := "rgba(38, 49, 72, 0.06)"
	if dark {
		fallback = "rgba(255, 255, 255, 0.08)"
	}
	return theme.Color(theme.SkinSemanticToken("", dark, "--dsw-alias-markdown-inline-code", fallback))
}

func inlineCodeFill(st Style) ui.Color {
	if st.InlineCodeBG.A != 0 {
		return st.InlineCodeBG
	}
	return InlineCodeBase(st.Dark)
}

// inlineCodeBorder is the chip's 0.5px --dsw-alias-border-l1 stroke.
func inlineCodeBorder(st Style) ui.Color {
	fallback := "rgba(0, 0, 0, 0.04)"
	if st.Dark {
		fallback = "rgba(255, 255, 255, 0.06)"
	}
	return theme.Color(theme.SkinSemanticToken("", st.Dark, "--dsw-alias-border-l1", fallback))
}

// inlineCodeRadius follows the chip's --dsw-radius-sm corner, 6px in the
// standard family.
func inlineCodeRadius(st Style) float32 {
	raw := theme.SkinSemanticToken("", st.Dark, "--dsw-radius-sm", "6px")
	if v, err := strconv.ParseFloat(strings.TrimSuffix(raw, "px"), 32); err == nil && v > 0 {
		return float32(v)
	}
	return 6
}

// inlineCodeInsetY centers the chip's 19px code line inside the body's
// 24px leading; the compact variant's inline code already fills its line.
func inlineCodeInsetY(st Style) float32 {
	if inset := (BodyLineHeight(st) - (InlineCodeSize(st) + 7)) / 2; inset > 0 {
		return inset
	}
	return 0
}

// tableHeaderFill resolves the table header fill.
func tableHeaderFill(st Style) ui.Color {
	if st.TableHeader.A != 0 {
		return st.TableHeader
	}
	if st.Muted.A != 0 {
		return st.Muted.Alpha(.08)
	}
	return ui.Transparent
}

// quoteBorderColor keeps a caller's rule, else the muted body color.
func quoteBorderColor(st Style) ui.Color {
	if st.QuoteBorder.A != 0 {
		return st.QuoteBorder
	}
	return st.Muted
}

// quoteInset is the blockquote text inset. The conversation passes the Original
// 14px; other document surfaces keep their 16px baseline.
func quoteInset(st Style) float32 {
	if st.Compact {
		return 8 // .compact blockquote
	}
	if st.QuoteInset > 0 {
		return st.QuoteInset
	}
	return 16
}

// codeLabel / codeCopyLabel / codeCopiedLabel / codeWrapLabels resolve the code
// card copy. Owners that localize pass them; everyone else keeps the Original
// English defaults.
func codeLabel(st Style) string {
	if st.CodeLabel != "" {
		return st.CodeLabel
	}
	return "code"
}

func codeCopyLabel(st Style) string {
	if st.CopyLabel != "" {
		return st.CopyLabel
	}
	return "Copy code"
}

func codeCopiedLabel(st Style) string {
	if st.CopiedLabel != "" {
		return st.CopiedLabel
	}
	return "Copied"
}

func codeWrapLabels(st Style) (wrap, noWrap string) {
	wrap, noWrap = st.WrapLabel, st.NoWrapLabel
	if wrap == "" {
		wrap = "Wrap lines"
	}
	if noWrap == "" {
		noWrap = "Unwrap lines"
	}
	return wrap, noWrap
}

// ruleBorderColor keeps a caller's rule, else the muted body color.
func ruleBorderColor(st Style) ui.Color {
	if st.RuleBorder.A != 0 {
		return st.RuleBorder
	}
	return st.Muted
}

// listMarkerColor mirrors MarkdownText.module.css li::marker (label-secondary);
// an unset Marker keeps the Muted fallback.
func listMarkerColor(st Style) ui.Color {
	if st.Marker != ui.Transparent {
		return st.Marker
	}
	return st.Muted
}
