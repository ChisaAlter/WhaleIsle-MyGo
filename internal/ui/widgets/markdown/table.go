package markdown

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// Table metrics from the Original MarkdownText sheet: 12px header, 6px cell
// padding, a 0.5px row rule and a horizontal scroll wrapper for wide tables.
const (
	tableCellPadY      = 6
	tableCellPadX      = 10
	tableMinColumn     = 88
	tableMaxColumn     = 260
	tableWeightMin     = 6
	tableWeightMax     = 40
	tableHeaderRowPadY = 5
)

// viewTable lays a GFM table out with content-proportional columns. The
// Original sizes columns from their content and scrolls the wrapper when the
// result exceeds the message column, so the native port keeps one Row per
// record with Grow weights plus a per-column pixel floor.
func viewTable(c *ui.Context, table *extast.Table, source string, st Style) {
	var rows []ast.Node
	var headerRow ast.Node
	columns := 0
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		if _, header := row.(*extast.TableHeader); header && headerRow == nil {
			headerRow = row
		}
		count := 0
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			count++
		}
		if count > columns {
			columns = count
		}
		rows = append(rows, row)
	}
	if columns == 0 || len(rows) == 0 {
		return
	}
	weights := make([]float32, columns)
	widths := make([]float32, columns)
	for _, row := range rows {
		index := 0
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if index >= columns {
				break
			}
			text := nodeText(cell, source)
			weight := float32(utf8.RuneCountInString(text))
			if weight < tableWeightMin {
				weight = tableWeightMin
			}
			if weight > tableWeightMax {
				weight = tableWeightMax
			}
			if weight > weights[index] {
				weights[index] = weight
			}
			width := float32(utf8.RuneCountInString(text))*7 + 2*tableCellPadX
			if width < tableMinColumn {
				width = tableMinColumn
			}
			if width > tableMaxColumn {
				width = tableMaxColumn
			}
			if width > widths[index] {
				widths[index] = width
			}
			index++
		}
	}
	for index := range weights {
		if weights[index] == 0 {
			weights[index] = tableWeightMin
		}
		if widths[index] == 0 {
			widths[index] = tableMinColumn
		}
	}
	var total float32
	for _, width := range widths {
		total += width
	}
	rule := st.Muted.Alpha(.28)
	ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).MinWidth(total).Children(func() {
			for _, row := range rows {
				_, isHeader := row.(*extast.TableHeader)
				fill := ui.Transparent
				padY := float32(tableCellPadY)
				if isHeader {
					fill = tableHeaderFill(st)
					padY = tableHeaderRowPadY
				}
				ui.Row(c).FillWidth().AlignItems(ui.Start).Background(fill).Draw(func(p *ui.Painter, r ui.Rect) {
					p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - .5, W: r.W, H: .5}, rule, 0)
				}).Children(func() {
					index := 0
					for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
						if index >= columns {
							break
						}
						column := index
						index++
						ui.Box(c).Grow(weights[column]).MinWidth(widths[column]).Padding(padY, tableCellPadX, padY, tableCellPadX).MinWidth(0).Children(func() {
							viewTableCell(c, cell, source, st, isHeader, tableAlignment(table, column))
						})
					}
				})
			}
		})
	})
}

// tableAlignment reads the GFM alignment marker for one column.
func tableAlignment(table *extast.Table, column int) extast.Alignment {
	if table.Alignments == nil || column >= len(table.Alignments) {
		return extast.AlignNone
	}
	return table.Alignments[column]
}

func viewTableCell(c *ui.Context, cell ast.Node, source string, st Style, header bool, align extast.Alignment) {
	cellStyle := st
	cellStyle.maths = map[int]mathFormula{}
	collectURLTargets(&cellStyle)
	cellStyle.textWeight = 0
	if header {
		cellStyle.textWeight = 600
	}
	cellStyle.lineHeight = BodyLineHeight(st)
	var blocks []richBlock
	cellStyle.blockAt = func(start, end int, node ast.Node) {
		blocks = append(blocks, richBlock{start: start, end: end, node: node})
	}
	targets := map[int]string{}
	if st.ResolveFile != nil && st.OnOpenFile != nil {
		cellStyle.codeSpanAt = func(index int, value string) {
			targets[index] = st.ResolveFile(value)
		}
	}
	var spans []ui.Span
	walkInlines(cell, source, cellStyle, &spans)
	if header {
		for i := range spans {
			spans[i].Weight = 600
		}
	}
	textAlign := ui.Start
	switch align {
	case extast.AlignRight:
		textAlign = ui.End
	case extast.AlignCenter:
		textAlign = ui.Center
	}
	if len(blocks) > 0 {
		viewBlocks(c, source, cellStyle, spans, targets, blocks)
		return
	}
	if len(spans) == 0 {
		return
	}
	el := styledRichText(c, spans, cellStyle)
	el.TextAlign(textAlign)
}

// nodeText is the plain source text under one node, used only to size columns.
func nodeText(node ast.Node, source string) string {
	var out strings.Builder
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch value := n.(type) {
		case *ast.Text:
			out.Write(value.Segment.Value([]byte(source)))
		case *ast.String:
			out.Write(value.Value)
		}
		return ast.WalkContinue, nil
	})
	return out.String()
}
