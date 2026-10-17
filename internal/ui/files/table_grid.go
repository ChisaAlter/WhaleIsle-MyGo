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
	cols := max(0, colEnd-page.Request.Column)
	rows := max(0, rowEnd-page.Request.Row)
	cells := map[string]workspacefs.XLSXCell{}
	for _, cell := range page.Cells {
		cells[cell.Address] = cell
	}
	blocked := tab.readLoading || tab.readBlocked
	ui.ScrollHorizontal(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Width(float32(cols)*140 + 48).FillHeight().MinHeight(0).Children(func() {
			ui.Row(c).Height(30).Children(func() {
				ui.Text(c, "#").Width(48)
				for col := page.Request.Column; col < colEnd; col++ {
					if ui.Button(c.Key(fmt.Sprintf("column-%d", col)), tableColumnName(col)).Width(140).Selected(tab.tableSelectionKind == "column" && tab.tableSelectionIndex == col).Disabled(blocked).Clicked() {
						tab.tableSelectionKind = "column"
						tab.tableSelectionIndex = col
						tab.tableCell = tableAddress(page.Request.Row, col)
					}
				}
			})
			tab.tableRows.Key = func(i int) any { return page.Request.Row + i }
			ui.List(c, &tab.tableRows, rows, func(i int) {
				row := page.Request.Row + i
				ui.Row(c).Height(36).Children(func() {
					if ui.Button(c, strconv.Itoa(row+1)).Width(48).Selected(tab.tableSelectionKind == "row" && tab.tableSelectionIndex == row).Disabled(blocked).Clicked() {
						tab.tableSelectionKind = "row"
						tab.tableSelectionIndex = row
						tab.tableCell = tableAddress(row, page.Request.Column)
					}
					for col := page.Request.Column; col < colEnd; col++ {
						address := tableAddress(row, col)
						cell := cells[address]
						button := ui.Button(c.Key(address), "").Width(140).Height(36).Padding(3, 6).Selected(tableSelected(tab, row, col)).Disabled(blocked).Label("Cell " + address).Tooltip(cell.DisplayValue)
						switch cell.Style.Vertical {
						case "top":
							button.AlignItems(ui.Start)
						case "bottom":
							button.AlignItems(ui.End)
						}
						button.Children(func() {
							text := ui.Text(c, cell.DisplayValue).FillWidth().SingleLine()
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
							tab.tableSelectionKind = "cell"
						}
					}
				})
			}).Grow(1).MinHeight(0)
		})
	})
	return true
}
