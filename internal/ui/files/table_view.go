package files

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	fonttheme "whaleisle/internal/ui/theme"
)

func requestTablePage(m Model, request workspacefs.XLSXPageRequest) {
	tab := selectedTab(m)
	if tab == nil || !isTablePreviewPath(tab.Path) {
		return
	}
	tab.tableRequest = request
	beginTextRead(m)
}

func paintTableControls(c *ui.Context, m Model, muted ui.Color) {
	tab := selectedTab(m)
	if tab == nil || !isTablePreviewPath(tab.Path) {
		return
	}
	ui.Text(c, label("只读 XLS/XLSX/CSV/TSV；每页最多 60 行 × 16 列。文件 ≤16 MiB，XLSX XML 单片 ≤8 MiB/合计 ≤32 MiB，单元格 ≤64 KiB，每页内容 ≤1 MiB。公式显示已存储值；常见日期、时间、百分比与数字格式参与显示，未支持格式保留原值。", "Read-only XLS/XLSX/CSV/TSV; up to 60 rows × 16 columns per page. File ≤16 MiB; XLSX XML ≤8 MiB/part and ≤32 MiB total; cell ≤64 KiB; page values ≤1 MiB. Formulas show stored values. Common date, time, percent and number formats are displayed; unsupported formats retain raw values.")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted).Padding(0, 12, 4, 12)
	page := tab.tablePage
	if page == nil {
		return
	}
	blocked := tab.readLoading || tab.readBlocked
	ui.Row(c).Wrap().Gap(6).Padding(0, 12, 4, 12).Children(func() {
		options := make([]string, 0, len(page.Sheets))
		selected := ""
		for i, sheet := range page.Sheets {
			option := fmt.Sprintf("%d. %s", i+1, sheet.Name)
			options = append(options, option)
			if sheet.ID == page.Request.Sheet {
				selected = option
			}
		}
		if ui.Select(c, &selected, options).Disabled(tab.readLoading).Changed() {
			for i, option := range options {
				if option == selected {
					requestTablePage(m, workspacefs.XLSXPageRequest{Sheet: page.Sheets[i].ID, Version: page.Version})
					break
				}
			}
		}
		if ui.Button(c, label("上一行页", "Previous rows")).Disabled(blocked || page.Request.Row == 0).Clicked() {
			req := page.Request
			req.Row = max(0, req.Row-req.Rows)
			requestTablePage(m, req)
		}
		if ui.Button(c, label("下一行页", "Next rows")).Disabled(blocked || page.Request.Row+page.Request.Rows >= page.TotalRows).Clicked() {
			req := page.Request
			req.Row += req.Rows
			requestTablePage(m, req)
		}
		if ui.Button(c, label("上一列页", "Previous columns")).Disabled(blocked || page.Request.Column == 0).Clicked() {
			req := page.Request
			req.Column = max(0, req.Column-req.Columns)
			requestTablePage(m, req)
		}
		if ui.Button(c, label("下一列页", "Next columns")).Disabled(blocked || page.Request.Column+page.Request.Columns >= page.TotalColumns).Clicked() {
			req := page.Request
			req.Column += req.Columns
			requestTablePage(m, req)
		}
		if ui.Button(c, label("复制本页原始 TSV", "Copy raw page TSV")).Disabled(blocked || page.Problem != "").Clicked() {
			mygo.Clipboard.WriteText(page.TSV)
		}
		if ui.Button(c, label("复制本页显示 TSV", "Copy displayed page TSV")).Disabled(blocked || page.Problem != "").Clicked() {
			mygo.Clipboard.WriteText(page.DisplayTSV)
		}
	})
	if len(page.Cells) > 0 {
		ui.Row(c).Wrap().Gap(6).Padding(0, 12, 4, 12).Children(func() {
			cells := make([]string, 0, len(page.Cells))
			for _, cell := range page.Cells {
				cells = append(cells, cell.Address)
			}
			found := false
			for _, address := range cells {
				if tab.tableCell == address {
					found = true
					break
				}
			}
			if !found {
				tab.tableCell = cells[0]
			}
			ui.Select(c, &tab.tableCell, cells).Disabled(blocked)
			if ui.Button(c, label("复制单元格显示值", "Copy displayed cell value")).Disabled(blocked).Clicked() {
				for _, cell := range page.Cells {
					if cell.Address == tab.tableCell {
						mygo.Clipboard.WriteText(cell.DisplayValue)
						break
					}
				}
			}
			if ui.Button(c, label("复制单元格原文", "Copy original cell value")).Disabled(blocked).Clicked() {
				for _, cell := range page.Cells {
					if cell.Address == tab.tableCell {
						mygo.Clipboard.WriteText(cell.Value)
						break
					}
				}
			}
		})
	}
}
