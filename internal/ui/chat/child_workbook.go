package chat

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func childWorkbookView(c *ui.Context, path string, read session.AgentResourceRead, opts ReadOnlyResourceOptions, st ToolCardStyle) {
	page := read.Table
	if page == nil {
		return
	}
	box := ui.Column(c.Key("child-workbook:" + path)).FillWidth().Gap(5)
	selectedCell := ui.Local(box, "cell", func() string { return "" })
	run := func(request workspacefs.XLSXPageRequest) {
		if opts.OnReadTable != nil {
			opts.OnReadTable(path, request)
		}
	}
	blocked := read.Busy || opts.OnReadTable == nil
	box.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatWorkbookReadOnly)).FontSize(10).TextColor(st.Muted)
		options := make([]string, len(page.Sheets))
		selected := ""
		for i, sheet := range page.Sheets {
			options[i] = fmt.Sprintf("%d. %s", i+1, sheet.Name)
			if sheet.ID == page.Request.Sheet {
				selected = options[i]
			}
		}
		if ui.Select(c, &selected, options).Disabled(blocked).Changed() {
			for i, option := range options {
				if option == selected {
					run(workspacefs.XLSXPageRequest{Sheet: page.Sheets[i].ID, Version: page.Version})
				}
			}
		}
		ui.Row(c).Wrap().Gap(5).Children(func() {
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookPrevRows), buttonGhost, blocked || page.Request.Row == 0).Clicked() {
				r := page.Request
				r.Row = max(0, r.Row-r.Rows)
				run(r)
			}
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookNextRows), buttonGhost, blocked || page.Request.Row+page.Request.Rows >= page.TotalRows).Clicked() {
				r := page.Request
				r.Row += r.Rows
				run(r)
			}
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookPrevCols), buttonGhost, blocked || page.Request.Column == 0).Clicked() {
				r := page.Request
				r.Column = max(0, r.Column-r.Columns)
				run(r)
			}
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookNextCols), buttonGhost, blocked || page.Request.Column+page.Request.Columns >= page.TotalColumns).Clicked() {
				r := page.Request
				r.Column += r.Columns
				run(r)
			}
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookCopyRawTSV), buttonGhost, blocked || page.Problem != "").Clicked() {
				c.Services().WriteClipboard(page.TSV)
			}
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookCopyShownTSV), buttonGhost, blocked || page.Problem != "").Clicked() {
				c.Services().WriteClipboard(page.DisplayTSV)
			}
		})
		ui.Text(c, page.Text).FontSize(12).TextColor(st.FG).Selectable()
		if page.Problem != "" {
			ui.Text(c, page.Problem).FontSize(11).TextColor(st.Muted)
		}
		if len(page.Cells) > 0 {
			options := make([]string, len(page.Cells))
			found := false
			for i, cell := range page.Cells {
				options[i] = cell.Address
				if cell.Address == *selectedCell {
					found = true
				}
			}
			if !found {
				*selectedCell = page.Cells[0].Address
			}
			ui.Select(c, selectedCell, options).Disabled(blocked)
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookCopyShown), buttonGhost, blocked).Clicked() {
				for _, cell := range page.Cells {
					if cell.Address == *selectedCell {
						c.Services().WriteClipboard(cell.DisplayValue)
						break
					}
				}
			}
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatWorkbookCopyRaw), buttonGhost, blocked).Clicked() {
				for _, cell := range page.Cells {
					if cell.Address == *selectedCell {
						c.Services().WriteClipboard(cell.Value)
					}
				}
			}
		}
	})
}
