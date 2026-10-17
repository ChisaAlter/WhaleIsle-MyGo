package files

import (
	"encoding/csv"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	"whaleisle/internal/desktop/workspacefs"
)

func tableColumnName(index int) string {
	result := ""
	for n := index + 1; n > 0; n = (n - 1) / 26 {
		result = string(rune('A'+(n-1)%26)) + result
	}
	return result
}
func tableAddress(row, column int) string { return tableColumnName(column) + strconv.Itoa(row+1) }
func tableCoordinates(address string) (row, column int, ok bool) {
	split := 0
	column = 0
	for split < len(address) && address[split] >= 'A' && address[split] <= 'Z' {
		column = column*26 + int(address[split]-'A') + 1
		split++
		if column > 16384 {
			return 0, 0, false
		}
	}
	row, err := strconv.Atoi(address[split:])
	return row - 1, column - 1, err == nil && split > 0 && row > 0
}
func tableAddressInPage(page *workspacefs.XLSXPage, address string) bool {
	row, col, ok := tableCoordinates(address)
	return ok && row >= page.Request.Row && row < min(page.TotalRows, page.Request.Row+page.Request.Rows) && col >= page.Request.Column && col < min(page.TotalColumns, page.Request.Column+page.Request.Columns)
}
func tableCellAt(page *workspacefs.XLSXPage, address string) workspacefs.XLSXCell {
	for _, cell := range page.Cells {
		if cell.Address == address {
			return cell
		}
	}
	return workspacefs.XLSXCell{Address: address}
}
func tableFormulaText(cell workspacefs.XLSXCell) string {
	if cell.HasFormula {
		if cell.Formula != "" {
			return "=" + strings.TrimPrefix(cell.Formula, "=")
		}
		return label("公式文本未提供（共享/数组公式）；仅显示已存储结果", "Formula text unavailable (shared/array formula); stored result only")
	}
	return cell.Value
}
func tableSelected(tab *FileTab, row, col int) bool {
	switch tab.tableSelectionKind {
	case "row":
		return row == tab.tableSelectionIndex
	case "column":
		return col == tab.tableSelectionIndex
	default:
		return tab.tableCell == tableAddress(row, col)
	}
}
func tableSelectionText(tab *FileTab, display bool) string {
	page := tab.tablePage
	if page == nil {
		return ""
	}
	if tab.tableSelectionKind != "row" && tab.tableSelectionKind != "column" {
		cell := tableCellAt(page, tab.tableCell)
		if display {
			return cell.DisplayValue
		}
		return cell.Value
	}
	rowEnd, colEnd := min(page.TotalRows, page.Request.Row+page.Request.Rows), min(page.TotalColumns, page.Request.Column+page.Request.Columns)
	var output strings.Builder
	writer := csv.NewWriter(&output)
	writer.Comma = '\t'
	for row := page.Request.Row; row < rowEnd; row++ {
		var cells []string
		for col := page.Request.Column; col < colEnd; col++ {
			if tableSelected(tab, row, col) {
				cell := tableCellAt(page, tableAddress(row, col))
				value := cell.Value
				if display {
					value = cell.DisplayValue
				}
				cells = append(cells, value)
			}
		}
		if len(cells) > 0 {
			_ = writer.Write(cells)
		}
	}
	writer.Flush()
	return strings.TrimSuffix(output.String(), "\n")
}
func paintTableFormulaBar(c *ui.Context, tab *FileTab) {
	page := tab.tablePage
	if !tableAddressInPage(page, tab.tableCell) {
		tab.tableCell = tableAddress(page.Request.Row, page.Request.Column)
		tab.tableSelectionKind = "cell"
	}
	cell := tableCellAt(page, tab.tableCell)
	formula := tableFormulaText(cell)
	ui.Row(c).Gap(6).Padding(4, 12).FillWidth().Children(func() {
		ui.Text(c, tab.tableCell+" · fx").Width(82)
		ui.TextInput(c, &formula).ReadOnly(true).Grow(1).MinWidth(0).Label("Read-only cell formula or value").Tooltip(formula)
		if ui.Button(c, label("复制公式/原文", "Copy formula/value")).Clicked() {
			mygo.Clipboard.WriteText(formula)
		}
	})
	ui.Row(c).Wrap().Gap(6).Padding(0, 12, 4, 12).Children(func() {
		format := cell.NumberFormat
		if format == "" {
			format = label("未提供", "not provided")
		}
		note := fmt.Sprintf("%s · %s", cell.Type, format)
		if cell.Style.Available {
			note += fmt.Sprintf(" · %s %.1f pt · %s/%s", cell.Style.FontName, cell.Style.FontSize, cell.Style.Horizontal, cell.Style.Vertical)
		} else {
			note += label(" · 字体/对齐未提供", " · font/alignment not provided")
		}
		if tablePaintUnsupported(cell.Style) {
			note += label(" · 部分源填充/边框无法显示，请用完整表格预览", " · Some source fill/border details are unsupported; use the full workbook viewer")
		}
		if !cell.FormatSupported && cell.Value != "" {
			note += label(" · 数字格式回退为原值", " · number format falls back to raw value")
		}
		ui.Text(c, note).FontSize(11).Grow(1)
		if ui.Button(c, label("复制已加载选区原值", "Copy loaded selection raw")).Clicked() {
			mygo.Clipboard.WriteText(tableSelectionText(tab, false))
		}
		if ui.Button(c, label("复制已加载选区显示值", "Copy loaded selection display")).Clicked() {
			mygo.Clipboard.WriteText(tableSelectionText(tab, true))
		}
	})
}
func paintTableGrid(c *ui.Context, m Model) bool {
	tab := selectedTab(m)
	if tab == nil || !isTablePreviewPath(tab.Path) || tab.tablePage == nil {
		return false
	}
	page := tab.tablePage
	if page.Problem != "" {
		ui.Text(c, page.Problem)
		return true
	}
	paintTableFormulaBar(c, tab)
	rowEnd, colEnd := min(page.TotalRows, page.Request.Row+page.Request.Rows), min(page.TotalColumns, page.Request.Column+page.Request.Columns)

	cells := map[string]workspacefs.XLSXCell{}
	for _, cell := range page.Cells {
		cells[cell.Address] = cell
	}
	blocked := tab.readLoading || tab.readBlocked
	hidden := page.DefaultRowsHidden
	for _, value := range page.HiddenRows {
		hidden = hidden || value
	}
	for _, value := range page.HiddenColumns {
		hidden = hidden || value
	}
	if hidden {
		ui.Text(c, label("源文件隐藏的行列未显示；本页原始复制仍保留其数据。", "Source-hidden rows and columns are not displayed; raw page copies retain their data.")).FontSize(10)
	}
	xs, ys := map[int]float32{}, map[int]float32{}
	width, height := float32(48), float32(0)
	for col := page.Request.Column; col < colEnd; col++ {
		xs[col] = width
		width += tableColumnWidth(page, col)
	}
	xs[colEnd] = width
	for row := page.Request.Row; row < rowEnd; row++ {
		ys[row] = height
		height += tableRowHeight(page, row)
	}
	ys[rowEnd] = height
	ui.ScrollHorizontal(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Width(width).FillHeight().MinHeight(0).Children(func() {
			ui.Row(c).Height(30).Children(func() {
				ui.Text(c, "#").Width(48)
				for col := page.Request.Column; col < colEnd; col++ {
					if tableColumnWidth(page, col) == 0 {
						continue
					}
					if ui.Button(c.Key(fmt.Sprintf("column-%d", col)), tableColumnName(col)).Width(tableColumnWidth(page, col)).Selected(tab.tableSelectionKind == "column" && tab.tableSelectionIndex == col).Disabled(blocked).Clicked() {
						tab.tableSelectionKind = "column"
						tab.tableSelectionIndex = col
						tab.tableCell = tableAddress(page.Request.Row, col)
					}
				}
			})
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
				ui.Box(c).Width(width).Height(height).Children(func() {
					for row := page.Request.Row; row < rowEnd; row++ {
						if tableRowHeight(page, row) == 0 {
							continue
						}
						if ui.Button(c.Key(fmt.Sprintf("row-%d", row)), strconv.Itoa(row+1)).Absolute().Left(0).Top(ys[row]).Width(48).Height(tableRowHeight(page, row)).Selected(tab.tableSelectionKind == "row" && tab.tableSelectionIndex == row).Disabled(blocked).Clicked() {
							tab.tableSelectionKind = "row"
							tab.tableSelectionIndex = row
							tab.tableCell = tableAddress(row, page.Request.Column)
						}
						for col := page.Request.Column; col < colEnd; col++ {
							if tableColumnWidth(page, col) == 0 {
								continue
							}
							merge, merged := tableVisibleMerge(page, row, col)
							firstRow, firstCol := tableFirstVisibleMergeCell(page, merge)
							if row != firstRow || col != firstCol {
								continue
							}
							lastRow, lastCol := min(merge.LastRow+1, rowEnd), min(merge.LastColumn+1, colEnd)
							address := tableAddress(row, col)
							sourceAddress := tableAddress(merge.FirstRow, merge.FirstColumn)
							cell := cells[sourceAddress]
							tooltip := cell.DisplayValue
							if merged {
								tooltip = sourceAddress + ":" + tableAddress(merge.LastRow, merge.LastColumn) + " · " + tooltip
								if !tableAddressInPage(page, sourceAddress) {
									cell.DisplayValue = label("合并区域起点在其他页", "Merged origin on another page")
									tooltip = cell.DisplayValue + " · " + sourceAddress
								}
							}
							selected := tableSelected(tab, row, col)
							if merged && tab.tableSelectionKind == "cell" {
								selected = tab.tableCell == sourceAddress
							}
							if tab.tableSelectionKind == "row" {
								selected = tab.tableSelectionIndex >= firstRow && tab.tableSelectionIndex < lastRow
							}
							if tab.tableSelectionKind == "column" {
								selected = tab.tableSelectionIndex >= firstCol && tab.tableSelectionIndex < lastCol
							}
							button := ui.Button(c.Key(address), "").Absolute().Left(xs[col]).Top(ys[row]).Width(xs[lastCol]-xs[col]).Height(ys[lastRow]-ys[row]).Padding(3, 6).Clip().Selected(selected).Disabled(blocked).Label("Cell " + address).Tooltip(tooltip)
							paintTableSourceStyle(button, cell.Style, selected)
							switch cell.Style.Vertical {
							case "top":
								button.AlignItems(ui.Start)
							case "bottom":
								button.AlignItems(ui.End)
							}
							button.Children(func() {
								text := ui.Text(c, cell.DisplayValue).FillWidth()
								if !cell.Style.Wrap {
									text.SingleLine()
								}
								if cell.Style.FontName != "" {
									text.Font(cell.Style.FontName)
								}
								if cell.Style.FontSize > 0 {
									text.FontSize(min(float32(32), max(float32(8), cell.Style.FontSize*4/3)))
								}
								if cell.Style.Bold {
									text.Bold()
								}
								if cell.Style.Italic {
									text.Italic()
								}
								switch cell.Style.Horizontal {
								case "center", "centerContinuous":
									text.TextAlign(ui.Center)
								case "right":
									text.TextAlign(ui.End)
								case "left":
									text.TextAlign(ui.Start)
								default:
									if cell.Type == "number" || cell.Type == "date" {
										text.TextAlign(ui.End)
									}
								}
							})
							if button.Clicked() {
								tab.tableCell = address
								if merged && tableAddressInPage(page, sourceAddress) {
									tab.tableCell = sourceAddress
								}
								tab.tableSelectionKind = "cell"
							}
						}
					}
				})
			})
		})
	})
	return true
}
