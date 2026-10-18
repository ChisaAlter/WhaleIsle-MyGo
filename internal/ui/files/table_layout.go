package files

import (
	"math"
	"whaleisle/internal/desktop/workspacefs"
)

// XLSX heights are points; widths are font-dependent character units. Without
// the source font's maximum-digit metric the width conversion is approximate.
func tableRowHeight(page *workspacefs.XLSXPage, row int) float32 {
	hidden, explicit := page.HiddenRows[row]
	if hidden || (!explicit && page.DefaultRowsHidden) {
		return 0
	}
	value := page.RowHeights[row]
	if value <= 0 {
		value = page.DefaultRowHeight
	}
	if value <= 0 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return 36
	}
	return min(float32(1024), max(float32(1), value*4/3))
}
func tableColumnWidth(page *workspacefs.XLSXPage, col int) float32 {
	if page.HiddenColumns[col] {
		return 0
	}
	value := page.ColumnWidths[col]
	if value <= 0 {
		value = page.DefaultColumnWidth
	}
	if value <= 0 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return 140
	}
	return min(float32(2048), max(float32(1), value*7+5))
}

// Each page paints only the intersection, never expands its loaded cell budget.
func tableVisibleMerge(page *workspacefs.XLSXPage, row, col int) (workspacefs.XLSXMerge, bool) {
	for _, merge := range page.Merges {
		if row >= merge.FirstRow && row <= merge.LastRow && col >= merge.FirstColumn && col <= merge.LastColumn {
			return merge, true
		}
	}
	return workspacefs.XLSXMerge{FirstRow: row, LastRow: row, FirstColumn: col, LastColumn: col}, false
}

func tableFirstVisibleMergeCell(page *workspacefs.XLSXPage, merge workspacefs.XLSXMerge) (int, int) {
	row, col := max(merge.FirstRow, page.Request.Row), max(merge.FirstColumn, page.Request.Column)
	rowEnd, colEnd := min(merge.LastRow+1, page.Request.Row+page.Request.Rows), min(merge.LastColumn+1, page.Request.Column+page.Request.Columns)
	for row < rowEnd && tableRowHeight(page, row) == 0 {
		row++
	}
	for col < colEnd && tableColumnWidth(page, col) == 0 {
		col++
	}
	return row, col
}
