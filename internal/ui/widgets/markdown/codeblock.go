package markdown

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"

	"whaleisle/internal/ui/theme"
)

// Code-card geometry from the Original CodeBlock sheet: a 30px sticky header,
// 11/19 source, 16px content padding and an 8px card radius.
const (
	codeCardRadius  = 8
	codeHeaderH     = 30
	codeToolbarBtn  = 22
	codeToolIcon    = 14
	codeBodyPadY    = 10
	codeBodyPadX    = 12
	codeCopiedDelay = time.Second
)

// viewCodeBlock draws one fenced code card: language label, wrap and copy
// actions in the header, and the highlighted source below. The toolbar mirrors
// the Original CodeToolbar (icon buttons with tooltips, a one-second copied
// check). RenderFence keeps owner-bound fences (OpenUI) untouched.
func viewCodeBlock(c *ui.Context, node ast.Node, source string, st Style) {
	language, code := codeBlockSource(node, source)
	// Tabs carry the source's nesting; expand them to the Original's four-column
	// tab stop so the card shows indentation instead of a flat left edge.
	code = strings.ReplaceAll(code, "\t", "    ")
	if st.RenderFence != nil && node.Lines().Len() > 0 && st.RenderFence(c, language, code, node.Lines().At(0).Start) {
		return
	}
	paintCodeCard(c, language, code, st)
}

func paintCodeCard(c *ui.Context, language, code string, st Style) {
	size := CodeSize(st)
	font := theme.RenderPreferences().Code.Family
	root := ui.Column(c).FillWidth().MinWidth(0).Background(st.CodeBG).Radius(codeCardRadius).Clip()
	wrap := ui.Local(root, "wrap-code", func() bool { return st.CodeWrapDefault })
	copied := ui.Local(root, "copied-until", func() time.Time { return time.Time{} })
	root.Children(func() {
		ui.Row(c).FillWidth().Height(codeHeaderH).Background(codeBanner(st)).
			Padding(0, 6, 0, 12).Gap(2).AlignItems(ui.Center).Children(func() {
			label := language
			if label == "" {
				label = codeLabel(st)
			}
			ui.Text(c, label).FontSize(size).TextColor(st.Muted).Font(font).SingleLine().Grow(1).MinWidth(0)
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
					c.WriteClipboard(code)
					*copied = time.Now().Add(codeCopiedDelay)
				})
			}
		})
		// A trailing newline would paint one empty line box; the clipboard keeps
		// the exact source, the card shows its lines.
		spans := highlightedCodeSpans(language, strings.TrimRight(code, "\n"), st)
		lineHeight := codeLineHeight(st)
		paint := func(el ui.Element) ui.Element {
			return el.Font(font).FontSize(size).FixedLineHeight(lineHeight).TextColor(st.FG)
		}
		if *wrap {
			paint(ui.RichText(c, spans...).FillWidth().MinWidth(0)).Padding(codeBodyPadY, codeBodyPadX, codeBodyPadY, codeBodyPadX)
		} else {
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
				paint(ui.RichText(c, spans...).NoWrap()).Padding(codeBodyPadY, codeBodyPadX, codeBodyPadY, codeBodyPadX)
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
		Justify(ui.Center).AlignItems(ui.Center)
	if b.Hovered() {
		b.Background(st.Muted.Alpha(.12)).TextColor(st.FG)
	}
	b.Children(func() { ui.Icon(c, icon).Width(codeToolIcon).Height(codeToolIcon).PassThrough() })
	if action != nil && b.Clicked() {
		action()
	}
}
