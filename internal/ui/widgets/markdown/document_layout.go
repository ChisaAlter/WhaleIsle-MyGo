package markdown

import (
	"strconv"
	"strings"

	"whaleisle/internal/markdownsyntax"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// Original MarkdownText.module.css and gradient-shadow-text.css keep document
// layout separate from chat surfaces. CSS vertical margins collapse by maximum.
func viewDocument(c *ui.Context, source string, st Style) {
	if strings.TrimSpace(source) == "" {
		return
	}
	viewStreamingDocument(c, source, st)
}

func documentMargins(node ast.Node) (top, bottom float32) {
	switch node := node.(type) {
	case *ast.Heading:
		if node.Level <= 3 {
			return 32, 16
		}
		return 16, 16
	case *ast.TextBlock:
		return 0, 0
	case *ast.Paragraph:
		if _, item := node.Parent().(*ast.ListItem); item {
			return 8, 8
		}
		return 16, 16
	case *ast.List:
		if _, item := node.Parent().(*ast.ListItem); item {
			return 4, 16
		}
		return 16, 16
	case *ast.ThematicBreak:
		return 32, 32
	case *ast.Blockquote:
		top = 16
		if child := node.FirstChild(); child != nil {
			first, _ := documentMargins(child)
			top = max(top, first)
		}
		if child := node.LastChild(); child != nil {
			_, bottom = documentMargins(child)
		}
		return top, bottom
	default:
		return 16, 16
	}
}

func viewDocumentBlocks(c *ui.Context, parent ast.Node, source string, st Style, outerCollapsed bool) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(0).Children(func() {
		var previous ast.Node
		var previousBottom float32
		for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
			top, bottom := documentMargins(node)
			gap := max(previousBottom, top)
			if previous == nil && outerCollapsed {
				gap = 0
			}
			if heading, ok := previous.(*ast.Heading); ok && heading.Level >= 4 {
				if _, list := node.(*ast.List); list {
					gap = 8 // The source explicitly tightens h4-h6 followed by a list.
				}
			}
			ui.Column(c).FillWidth().MinWidth(0).Padding(gap, 0, 0, 0).Children(func() {
				viewDocumentBlock(c, node, source, st)
			})
			previous, previousBottom = node, bottom
		}
		if !outerCollapsed && previousBottom > 0 {
			ui.Box(c).Height(previousBottom)
		}
	})
}

func viewDocumentBlock(c *ui.Context, node ast.Node, source string, st Style) {
	switch node := node.(type) {
	case *ast.Heading:
		delta := st.FontSize - 14
		if node.Level <= 3 {
			st.FontSize = []float32{21, 19, 18}[node.Level-1] + delta
			st.lineHeight = []float32{30, 28, 26}[node.Level-1] + delta
			st.textWeight = 700
			if node.Level == 3 {
				if st.DocumentH3FontSize > 0 {
					st.FontSize = st.DocumentH3FontSize
				}
				if st.DocumentH3LineHeight > 0 {
					st.lineHeight = st.DocumentH3LineHeight
				}
			}
		} else {
			st.textWeight = 600
		}
		viewDocumentInlines(c, node, source, st)
	case *ast.Paragraph, *ast.TextBlock:
		viewDocumentInlines(c, node, source, st)
	case *ast.Blockquote:
		border := quoteBorderColor(st)
		inset := quoteInset(st)
		// Original MarkdownText blockquote: 2px rule with a text inset.
		ui.Column(c).FillWidth().MinWidth(0).Padding(0, 0, 0, inset).Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 2, H: r.H}, border, 0)
		}).Children(func() { viewDocumentBlocks(c, node, source, st, true) })
	case *ast.List:
		index := node.Start
		for item := node.FirstChild(); item != nil; item = item.NextSibling() {
			marker := "•"
			if node.IsOrdered() {
				marker = strconv.Itoa(index) + "."
				index++
			}
			top := float32(0)
			if item.PreviousSibling() != nil {
				top = 6 // li:not(:first-child)
			}
			ui.Row(c).FillWidth().MinWidth(0).Gap(0).Padding(top, 0, 0, 0).AlignItems(ui.Start).Children(func() {
				ui.Text(c, marker).Width(18).NoWrap().FontSize(st.FontSize).FixedLineHeight(BodyLineHeight(st)).TextColor(st.Muted)
				ui.Column(c).Grow(1).MinWidth(0).Children(func() { viewDocumentBlocks(c, item, source, st, true) })
			})
		}
	case *markdownsyntax.ImagePlaceholder:
		if st.RenderImage == nil || !st.RenderImage(c, node.Destination, "") {
			styledRichText(c, []ui.Span{{Text: "[image] " + node.Destination}}, st)
		}
	case *extast.FootnoteList:
		viewFootnotes(c, node, source, st)
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		viewCodeBlock(c, node, source, st)
	case *extast.Table:
		viewTable(c, node, source, st)
	case *blockMath:
		st.maths = map[int]mathFormula{}
		var spans []ui.Span
		if node.closed {
			appendFormula(node.formula, st, &spans)
		} else {
			appendPlainSpan(node.formula.Raw, st, &spans)
		}
		viewSpans(c, st, spans, nil, 0)
	case *ast.HTMLBlock:
		if st.LiteralHTML {
			styledRichText(c, []ui.Span{{Text: string(node.Text([]byte(source)))}}, st)
		}
	case *ast.ThematicBreak:
		border := st.RuleBorder
		if border.A == 0 {
			border = c.Theme().Border
		}
		ui.Box(c).Label("Markdown thematic break").FillWidth().Height(.5).Background(border)
	default:
		viewDocumentBlocks(c, node, source, st, true)
	}
}

func viewDocumentInlines(c *ui.Context, node ast.Node, source string, st Style) {
	key := makeBodyKey(st)
	if st.docInlines != nil {
		if cached, ok := st.docInlines[node]; ok && cached.key == key && cached.source == source {
			st.maths = cached.maths
			st.urlTargets = cached.urlTargets
			if len(cached.blocks) > 0 {
				viewBlocks(c, source, st, cached.spans, cached.targets, cached.blocks)
			} else {
				viewSpans(c, st, cached.spans, cached.targets, 0)
			}
			return
		}
	}
	st.maths = map[int]mathFormula{}
	collectURLTargets(&st)
	targets := map[int]string{}
	collectFileTargets(&st, targets)
	var blocks []richBlock
	st.blockAt = func(start, end int, node ast.Node) {
		blocks = append(blocks, richBlock{start: start, end: end, node: node})
	}
	var spans []ui.Span
	walkInlines(node, source, st, &spans)
	if st.docInlines != nil {
		st.docInlines[node] = cachedDocumentInlines{
			source: source, key: key, spans: spans, blocks: blocks,
			targets: targets, maths: st.maths, urlTargets: st.urlTargets,
		}
	}
	if len(blocks) > 0 {
		viewBlocks(c, source, st, spans, targets, blocks)
	} else {
		viewSpans(c, st, spans, targets, 0)
	}
}
