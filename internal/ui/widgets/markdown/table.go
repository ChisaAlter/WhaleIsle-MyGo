package markdown

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"

	"whaleisle/internal/ui/theme"
)

// Table metrics from the Original MarkdownText sheet: a 0.5px row and column
// rule, 6px cell padding, a 60 DIP column floor and a 40% ceiling of the
// message column. Widths come from measured content, not a rune guess: CJK
// cells are twice as wide as the Latin average, and a proportional split used
// to wrap them.
const (
	tableCellPadY      = 6
	tableCellPadX      = 10
	tableMinColumn     = 60
	tableColumnShare   = 0.4
	tableHeaderRowPadY = 5
	tableDefaultAxis   = 748
)

// viewTable lays a GFM table out from measured cell content. Every column takes
// its widest cell plus padding, clamped to [60, 40% of the available width]; the
// table only scrolls horizontally when those widths no longer fit.
func viewTable(c *ui.Context, table *extast.Table, source string, st Style) {
	var rows []ast.Node
	columns := 0
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
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
	available := tableAxis(st)
	maxColumn := available * tableColumnShare
	if maxColumn < tableMinColumn {
		maxColumn = tableMinColumn
	}
	widths := make([]float32, columns)
	alignments := make([]extast.Alignment, columns)
	for column := 0; column < columns; column++ {
		alignments[column] = tableAlignment(table, column)
	}
	for _, row := range rows {
		_, isHeader := row.(*extast.TableHeader)
		index := 0
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if index >= columns {
				break
			}
			width := measureCellWidth(c, cell, source, st, isHeader) + 2*tableCellPadX
			width = clampf(width, tableMinColumn, maxColumn)
			if width > widths[index] {
				widths[index] = width
			}
			index++
		}
	}
	var total float32
	for index := range widths {
		if widths[index] == 0 {
			widths[index] = tableMinColumn
		}
		total += widths[index]
	}
	paint := func(ctx *ui.Context) {
		ui.Column(ctx).Width(total).MinWidth(0).Children(func() {
			for _, row := range rows {
				paintTableRow(ctx, row, source, st, widths, alignments, columns, total)
			}
		})
	}
	if total > available {
		ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() { paint(c) })
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Children(func() { paint(c) })
}

// tableAxis is the width the table may use before it scrolls. The conversation
// passes its content axis; a document without one keeps the Original 748.
func tableAxis(st Style) float32 {
	if st.AvailableWidth > 0 {
		return st.AvailableWidth
	}
	return tableDefaultAxis
}

// paintTableRow draws one record: the header fill, a 0.5px bottom rule and the
// column separators, so a scrolled table keeps its grid.
func paintTableRow(c *ui.Context, row ast.Node, source string, st Style, widths []float32, alignments []extast.Alignment, columns int, total float32) {
	_, isHeader := row.(*extast.TableHeader)
	fill := ui.Transparent
	padY := float32(tableCellPadY)
	if isHeader {
		fill = tableHeaderFill(st)
		padY = tableHeaderRowPadY
	}
	// A 0.5 DIP rule disappears on integer-pixel software rendering, so the grid
	// uses a 1px hairline at a low alpha and stays visible either way.
	rule := st.Muted.Alpha(.20)
	ui.Row(c).Width(total).MinWidth(0).AlignItems(ui.Start).Background(fill).Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, rule, 0)
	}).Children(func() {
		index := 0
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if index >= columns {
				break
			}
			column := index
			index++
			width := widths[column]
			last := column == columns-1
			box := ui.Box(c).Width(width).MinWidth(0).Padding(padY, tableCellPadX, padY, tableCellPadX)
			if !last {
				box.Draw(func(p *ui.Painter, r ui.Rect) {
					p.Fill(ui.Rect{X: r.X + r.W - 1, Y: r.Y + padY/2, W: 1, H: r.H - padY}, rule, 0)
				})
			}
			box.Children(func() {
				viewTableCell(c, cell, source, st, isHeader, alignments[column])
			})
		}
	})
}

// measureCellWidth runs the cell's plain text through the same font and size
// the cell paints with, so a CJK column gets its true advance width.
func measureCellWidth(c *ui.Context, cell ast.Node, source string, st Style, header bool) float32 {
	text := nodeText(cell, source)
	if text == "" {
		return 0
	}
	weight := 400
	if header {
		weight = 600
	}
	span := ui.Span{Text: text, Size: st.FontSize, Weight: weight, Font: theme.RenderPreferences().Chat.Family}
	width, _ := c.MeasureText(0, span)
	return width
}

func clampf(value, low, high float32) float32 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
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

var _ = utf8.RuneCountInString
