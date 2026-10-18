package files

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestTableSavedLayoutMergeAndSelection(t *testing.T) {
	m, tab := tableGridFixture()
	page := tab.tablePage
	page.Merges = []workspacefs.XLSXMerge{{FirstRow: 0, LastRow: 1, FirstColumn: 0, LastColumn: 1}}
	page.RowHeights = map[int]float32{0: 30, 1: 45}
	page.ColumnWidths = map[int]float32{0: 20, 1: 30}
	page.Cells[0].Style.Wrap = true
	if tableRowHeight(page, 0) != 40 || tableColumnWidth(page, 1) != 215 {
		t.Fatal("source dimensions ignored")
	}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { paintTableGrid(c, m) }) }, 700, 400)
	if err := tt.Click("Cell A1"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Cell B2"); err == nil {
		t.Fatal("covered cell remains independently clickable")
	}
	if tab.tableCell != "A1" || tableSelectionText(tab, false) != "0.25" || tableFormulaText(tableCellAt(page, tab.tableCell)) != "=B1/100" {
		t.Fatal("merge lost origin selection")
	}
	if err := tt.Click("B"); err != nil {
		t.Fatal(err)
	}
	if tab.tableSelectionKind != "column" {
		t.Fatal("header selection lost")
	}
	page.Request.Row = 1
	merge, ok := tableVisibleMerge(page, 1, 1)
	if !ok || merge.FirstRow != 0 {
		t.Fatal("page clipping lost source origin")
	}
}
