package markdown

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"

	"whaleisle/internal/ui/theme"
)

// Code-card geometry from the Original CodeCard sheet: a 42px banner row
// (10/8 inset around 24px action seats, gap 12 to the 4-gap action cluster),
// 11/19 source, a 6/22/20 content inset and the theme's --dsw-radius-lg corner.
const (
	codeHeaderH       = 42
	codeToolbarBtn    = 24
	codeToolIcon      = 14
	codeBodyPadTop    = 6
	codeBodyPadBottom = 20
	codeBodyPadX      = 22
	codeCopiedDelay   = time.Second
)

// Plain-banner geometry from the Original CodeBlock sheet used when the owner
// passes no toolbar labels: a 36px .banner row (9/14 inset around the 11/18
// infostring and text copy action) and a uniform 16px pre inset that always
// wraps.
const (
	codeBannerH     = 36
	codeBannerPadX  = 14
	codeBannerPadY  = 9
	codePlainBodyPx = 16
)

// codeCardRadius follows the card's --dsw-radius-lg corner, which the theme
// families set between 10 and 12px.
func codeCardRadius(dark bool) float32 {
	raw := theme.SkinSemanticToken("", dark, "--dsw-radius-lg", "8px")
	if v, err := strconv.ParseFloat(strings.TrimSuffix(raw, "px"), 32); err == nil && v > 0 {
		return float32(v)
	}
	return 8
}

// viewCodeBlock draws one fenced code card: language label, wrap and copy
// actions in the header, and the highlighted source below. The toolbar mirrors
// the Original CodeToolbar (icon buttons with tooltips, a one-second copied
// check). RenderFence keeps owner-bound fences (OpenUI) untouched.
func viewCodeBlock(c *ui.Context, node ast.Node, source string, st Style) {
	language, code := codeBlockSource(node, source)
	if language == "math" && !st.Streaming && strings.TrimSpace(code) != "" {
		mathStyle := st
		mathStyle.maths = map[int]mathFormula{}
		var spans []ui.Span
		appendFormula(mathFormula{TeX: code, Raw: code, Display: true}, mathStyle, &spans)
		viewSpans(c, mathStyle, spans, nil, 0)
		return
	}
	// Keep parsed source bytes for copying; expanding tabs is display-only.
	clipboardCode := code
	// Tabs carry the source's nesting; expand them to the Original's four-column
	// tab stop so the card shows indentation instead of a flat left edge.
	code = strings.ReplaceAll(code, "\t", "    ")
	if st.RenderFence != nil && node.Lines().Len() > 0 && st.RenderFence(c, language, code, node.Lines().At(0).Start) {
		return
	}
	paintCodeCard(c, language, code, clipboardCode, st)
}

func paintCodeCard(c *ui.Context, language, code, clipboardCode string, st Style) {
	size := CodeSize(st)
	font := theme.RenderPreferences().Code.Family
	root := ui.Column(c).FillWidth().MinWidth(0).Background(st.CodeBG).Radius(codeCardRadius(st.Dark)).Clip().Label("Code block")
	wrap := ui.Local(root, "wrap-code", func() bool { return st.CodeWrapDefault })
	copied := ui.Local(root, "copied-until", func() time.Time { return time.Time{} })
	highlight := ui.Local(root, "highlight", func() codeHighlightCache { return codeHighlightCache{} })
	root.Children(func() {
		if st.PlainCodeBanner {
			// Original CodeBlock .banner: infostring stays empty for plain
			// fences, the text copy action inherits the banner's 11/18 font.
			ui.Row(c).FillWidth().Height(codeBannerH).Unselectable().Background(codeBanner(st)).Label("Code block banner").
				Padding(codeBannerPadY, codeBannerPadX, codeBannerPadY, codeBannerPadX).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, language).FontSize(size).TextColor(st.FG).Font(font).FixedLineHeight(18).SingleLine().Grow(1).MinWidth(0)
				bannerLabel := codeCopyLabel(st)
				if time.Now().Before(*copied) {
					bannerLabel = codeCopiedLabel(st)
				}
				if ui.Button(c, bannerLabel).Label(bannerLabel).FontSize(size).FixedLineHeight(18).TextColor(st.FG).
					Padding(0).BorderWidth(0).Background(ui.Transparent).Cursor(ui.CursorPointer).Shrink(0).Clicked() {
					c.WriteClipboard(clipboardCode)
					*copied = time.Now().Add(codeCopiedDelay)
				}
			})
		} else {
			ui.Row(c).FillWidth().Height(codeHeaderH).Unselectable().Background(codeBanner(st)).Label("Code block banner").
				Padding(10, 18, 8, 22).Gap(12).AlignItems(ui.Center).Children(func() {
				label := language
				if label == "" {
					label = codeLabel(st)
				}
				ui.Text(c, label).FontSize(size).TextColor(st.Muted).Font(font).FixedLineHeight(18).SingleLine().Grow(1).MinWidth(0)
				ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Children(func() {
					// The wrap seat reports the action it performs: wrapped → "preserve
					// columns", unwrapped → "wrap lines" (Original CodeToolbar).
					wrapText, noWrapText := codeWrapLabels(st)
					wrapIcon, wrapLabel := codeIconWrap, wrapText
					if *wrap {
						wrapIcon, wrapLabel = codeIconNoWrap, noWrapText
					}
					codeToolbarButton(c, wrapIcon, wrapLabel, st, func() { *wrap = !*wrap })
					if time.Now().Before(*copied) {
						codeToolbarButton(c, codeIconCheck, codeCopiedLabel(st), st, nil)
					} else {
						codeToolbarButton(c, codeIconCopy, codeCopyLabel(st), st, func() {
							c.WriteClipboard(clipboardCode)
							*copied = time.Now().Add(codeCopiedDelay)
						})
					}
				})
			})
		}
		// A trailing newline would paint one empty line box; the clipboard keeps
		// the exact source, the card shows its lines.
		spans := highlight.update(language, code, st)
		lineHeight := codeLineHeight(st)
		paint := func(el ui.Element) ui.Element {
			return el.Font(font).FontSize(size).FixedLineHeight(lineHeight).TextColor(st.FG).Selectable()
		}
		if st.PlainCodeBanner {
			// The plain .block pre wraps unconditionally at a 16px inset.
			paint(ui.RichText(c, spans...).FillWidth().MinWidth(0)).Padding(codePlainBodyPx, codePlainBodyPx, codePlainBodyPx, codePlainBodyPx)
		} else if *wrap {
			paint(ui.RichText(c, spans...).FillWidth().MinWidth(0)).Padding(codeBodyPadTop, codeBodyPadX, codeBodyPadBottom, codeBodyPadX)
		} else {
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
				paint(ui.RichText(c, spans...).NoWrap()).Padding(codeBodyPadTop, codeBodyPadX, codeBodyPadBottom, codeBodyPadX)
			})
		}
	})
	if time.Now().Before(*copied) {
		root.Draw(func(p *ui.Painter, _ ui.Rect) {
			if time.Now().After(*copied) {
				c.Services().Invalidate()
				return
			}
			p.After(time.Until(*copied))
		})
	}
}

// codeToolbarButton is one 22px icon seat in the code toolbar. A nil action
// renders the copied state, which stays visually pressed without a handler.
func codeToolbarButton(c *ui.Context, icon *ui.SVG, label string, st Style, action func()) {
	b := ui.Button(c, "").Label(label).Tooltip(label).Width(codeToolbarBtn).Height(codeToolbarBtn).
		Padding(0).Radius(4).BorderWidth(0).Background(ui.Transparent).TextColor(st.Muted).
		Cursor(ui.CursorPointer).
		Justify(ui.Center).AlignItems(ui.Center)
	if b.Hovered() {
		hover := st.ToolCodeHover
		if hover.A == 0 {
			hover = st.Muted.Alpha(.12)
		}
		b.Background(hover).TextColor(st.FG)
	}
	b.Children(func() { ui.Icon(c, icon).Width(codeToolIcon).Height(codeToolIcon).PassThrough() })
	if action != nil && b.Clicked() {
		action()
	}
}
