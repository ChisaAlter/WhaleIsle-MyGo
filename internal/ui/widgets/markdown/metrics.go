package markdown

import "github.com/egoist/mygo/ui"

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

// BodyLineHeight is the base 24px leading at the 14px default body size.
func BodyLineHeight(st Style) float32 {
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
// 14px default.
func InlineCodeSize(st Style) float32 {
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

// codeBanner resolves the code-card header fill. The Original banner is
// --dsw-alias-markdown-code-block-banner, a muted tint over the body surface.
func codeBanner(st Style) ui.Color {
	if st.CodeBanner.A != 0 {
		return st.CodeBanner
	}
	if st.Muted.A != 0 {
		return st.Muted.Alpha(.10)
	}
	return ui.Transparent
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
