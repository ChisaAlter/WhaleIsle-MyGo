package files

import (
	"encoding/hex"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/workspacefs"
)

func tablePaintColor(value string) (ui.Color, bool) {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		return ui.Color{}, false
	}
	data, err := hex.DecodeString(value)
	if err != nil {
		return ui.Color{}, false
	}
	return ui.RGB(data[0], data[1], data[2]), true
}
func tableBorderPattern(style string) (float32, []float32, bool) {
	switch style {
	case "hair":
		return .5, nil, true
	case "thin":
		return 1, nil, true
	case "medium":
		return 2, nil, true
	case "thick":
		return 3, nil, true
	case "double":
		return 1, nil, true
	case "dashed":
		return 1, []float32{3, 2}, true
	case "mediumDashed":
		return 2, []float32{3, 2}, true
	case "dotted":
		return 1, []float32{1, 2}, true
	case "dashDot":
		return 1, []float32{3, 2, 1, 2}, true
	case "mediumDashDot", "slantDashDot":
		return 2, []float32{3, 2, 1, 2}, style != "slantDashDot"
	case "dashDotDot":
		return 1, []float32{3, 2, 1, 2, 1, 2}, true
	case "mediumDashDotDot":
		return 2, []float32{3, 2, 1, 2, 1, 2}, true
	}
	return 0, nil, false
}

// tableFillPattern maps the OOXML pattern fills that have a direct native
// Stripes equivalent. Cross-hatch and gray patterns intentionally return false:
// pretending a single stripe direction is equivalent would hide source paint
// differences. The widths are deterministic fallbacks for the OOXML dark/light
// density families; the saved foreground/background colours are never guessed.
func tableFillPattern(style workspacefs.XLSXCellStyle) (ui.Color, ui.Color, float32, float32, float32, bool) {
	if style.FillPattern == "" || style.FillPattern == "none" || style.FillPattern == "solid" {
		return ui.Color{}, ui.Color{}, 0, 0, 0, false
	}
	background, backgroundOK := tablePaintColor(style.FillColor)
	foreground, foregroundOK := tablePaintColor(style.FillColor2)
	if !backgroundOK || !foregroundOK {
		return ui.Color{}, ui.Color{}, 0, 0, 0, false
	}
	width, gap, angle := float32(1), float32(3), float32(0)
	switch style.FillPattern {
	case "lightVertical":
		angle = 0
	case "lightHorizontal":
		angle = 90
	case "lightDown":
		angle = 45
	case "lightUp":
		angle = 135
	case "darkVertical":
		width, gap, angle = 3, 2, 0
	case "darkHorizontal":
		width, gap, angle = 3, 2, 90
	case "darkDown":
		width, gap, angle = 3, 2, 45
	case "darkUp":
		width, gap, angle = 3, 2, 135
	default:
		return ui.Color{}, ui.Color{}, 0, 0, 0, false
	}
	return background, foreground, width, gap, angle, true
}
func tablePaintUnsupported(style workspacefs.XLSXCellStyle) bool {
	if style.PaintUnsupported {
		return true
	}
	if style.FillPattern != "" && style.FillPattern != "none" && style.FillPattern != "solid" {
		if _, _, _, _, _, ok := tableFillPattern(style); !ok {
			return true
		}
	}
	for _, edge := range style.Borders {
		if edge.Style != "" && edge.Style != "none" {
			if _, _, ok := tableBorderPattern(edge.Style); !ok {
				return true
			}
		}
	}
	return false
}
func paintTableSourceStyle(button ui.Element, style workspacefs.XLSXCellStyle, selected bool) {
	paintTableSourceFill(button, style, selected)
	button.DrawOver(func(p *ui.Painter, r ui.Rect) {
		paintTableBorderEdges(p, r, style.Borders)
	})
}

// paintTableSourceFill paints the saved cell fill while keeping the native
// selection highlight visible. It is split out so merged cells can retain
// their anchor fill while drawing the outline from each covered cell's saved
// border contribution.
func paintTableSourceFill(button ui.Element, style workspacefs.XLSXCellStyle, selected bool) {
	// Selection retains the native highlight instead of hiding it under source fill.
	if !selected {
		if style.FillPattern == "solid" {
			if color, ok := tablePaintColor(style.FillColor); ok {
				button.Background(color)
			}
		} else if background, foreground, width, gap, angle, ok := tableFillPattern(style); ok {
			button.Background(background).Stripes(foreground, width, gap, angle)
		}
	}
}

// paintTableBorderEdges paints one cell's four saved border edges. Edges are
// intentionally kept as independent lines so the same primitive can paint
// the per-cell segments around a merged region.
func paintTableBorderEdges(p *ui.Painter, r ui.Rect, borders [4]workspacefs.XLSXBorder) {
	for side, edge := range borders {
		paintTableBorderSegment(p, r, side, edge, 0, 0, 0)
	}
}

// paintTableBorderSegment paints one edge over [start,start+length]. A zero
// length means the whole corresponding side. The offset is measured along the
// perpendicular axis and is only used by merged-cell segments.
func paintTableBorderSegment(p *ui.Painter, r ui.Rect, side int, edge workspacefs.XLSXBorder, start, length, offset float32) {
	width, pattern, supported := tableBorderPattern(edge.Style)
	color, ok := tablePaintColor(edge.Color)
	if !ok || !supported {
		return
	}
	horizontal := side == 0 || side == 2
	if length <= 0 {
		if horizontal {
			length = r.W
		} else {
			length = r.H
		}
	}
	x, y := r.X, r.Y
	if horizontal {
		x += start
		if side == 0 {
			y += width / 2
		} else {
			y += r.H - width/2
		}
	} else {
		y += start
		if side == 1 {
			x += r.W - width/2
		} else {
			x += width / 2
		}
	}
	line := func(segmentStart, segmentEnd, segmentOffset float32) {
		if horizontal {
			p.Line(x+segmentStart, y+segmentOffset, x+segmentEnd, y+segmentOffset, width, color)
		} else {
			p.Line(x+segmentOffset, y+segmentStart, x+segmentOffset, y+segmentEnd, width, color)
		}
	}
	if len(pattern) == 0 {
		line(0, length, offset)
	} else {
		position, index := float32(0), 0
		for position < length {
			next := min(length, position+pattern[index%len(pattern)]*width)
			if index%2 == 0 {
				line(position, next, offset)
			}
			position = next
			index++
		}
	}
	if edge.Style == "double" {
		doubleOffset := float32(2)
		if side == 1 || side == 2 {
			doubleOffset = -2
		}
		line(0, length, offset+doubleOffset)
	}
}

// paintTableMergedSourceStyle preserves the anchor's fill but takes border
// segments from cells on the merged region's outer edges. The original
// workbook surface records borders for every cell, so covered cells may carry
// a different edge style/color than the anchor. Missing covered edges fall
// back to the anchor edge, retaining the existing outline when a workbook only
// stores the merged anchor style.
func paintTableMergedSourceStyle(button ui.Element, style workspacefs.XLSXCellStyle, selected bool, page *workspacefs.XLSXPage, merge workspacefs.XLSXMerge, cells map[string]workspacefs.XLSXCell, firstRow, firstCol, lastRow, lastCol int) {
	paintTableSourceFill(button, style, selected)
	button.DrawOver(func(p *ui.Painter, r ui.Rect) {
		// Draw top/bottom as one segment per visible column, and left/right as
		// one segment per visible row. Hidden dimensions have zero size.
		for col := max(merge.FirstColumn, page.Request.Column); col <= min(merge.LastColumn, lastCol-1); col++ {
			width := tableColumnWidth(page, col)
			if width <= 0 {
				continue
			}
			start := tableMergeOffsetX(page, firstCol, col)
			top := tableMergeEdge(cells, merge, 0, merge.FirstRow, col, style.Borders[0])
			bottom := tableMergeEdge(cells, merge, 2, merge.LastRow, col, style.Borders[2])
			paintTableBorderSegment(p, r, 0, top, start, width, 0)
			paintTableBorderSegment(p, r, 2, bottom, start, width, 0)
		}
		for row := max(merge.FirstRow, page.Request.Row); row <= min(merge.LastRow, lastRow-1); row++ {
			height := tableRowHeight(page, row)
			if height <= 0 {
				continue
			}
			start := tableMergeOffsetY(page, firstRow, row)
			left := tableMergeEdge(cells, merge, 3, row, merge.FirstColumn, style.Borders[3])
			right := tableMergeEdge(cells, merge, 1, row, merge.LastColumn, style.Borders[1])
			paintTableBorderSegment(p, r, 3, left, start, height, 0)
			paintTableBorderSegment(p, r, 1, right, start, height, 0)
		}
	})
}

func tableMergeOffsetX(page *workspacefs.XLSXPage, firstCol, col int) float32 {
	offset := float32(0)
	for current := firstCol; current < col; current++ {
		offset += tableColumnWidth(page, current)
	}
	return offset
}

func tableMergeOffsetY(page *workspacefs.XLSXPage, firstRow, row int) float32 {
	offset := float32(0)
	for current := firstRow; current < row; current++ {
		offset += tableRowHeight(page, current)
	}
	return offset
}

func tableMergeEdge(cells map[string]workspacefs.XLSXCell, merge workspacefs.XLSXMerge, side, row, col int, fallback workspacefs.XLSXBorder) workspacefs.XLSXBorder {
	if row < merge.FirstRow || row > merge.LastRow || col < merge.FirstColumn || col > merge.LastColumn {
		return fallback
	}
	cell, ok := cells[tableAddress(row, col)]
	if !ok {
		return fallback
	}
	edge := cell.Style.Borders[side]
	if edge.Style == "" || edge.Style == "none" {
		return fallback
	}
	return edge
}
