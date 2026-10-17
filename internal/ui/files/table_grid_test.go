package files

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func tableGridFixture() (Model, *FileTab) {
	m, tab := resourceModel()
	tab.Path = "a.xlsx"
	*m.SelectedFile = tab.Path
	m.Tabs.Workspaces["/remote"].Active = tab.Path
	tab.tablePage = &workspacefs.XLSXPage{Request: workspacefs.XLSXPageRequest{Rows: 2, Columns: 2}, TotalRows: 2, TotalColumns: 2, Cells: []workspacefs.XLSXCell{{Address: "A1", Value: "0.25", DisplayValue: "25%", HasFormula: true, Formula: "B1/100", NumberFormat: "0%"}, {Address: "B1", Value: "25", DisplayValue: "25"}, {Address: "A2", Value: "text", DisplayValue: "text"}}}
	return m, tab
}
func TestTableGridSelectionAndFormula(t *testing.T) {
	_, tab := tableGridFixture()
	tab.tableCell = "A1"
	if tableFormulaText(tableCellAt(tab.tablePage, "A1")) != "=B1/100" {
		t.Fatal("formula bar lost source")
	}
	if !strings.Contains(tableFormulaText(workspacefs.XLSXCell{HasFormula: true}), "共享") && !strings.Contains(tableFormulaText(workspacefs.XLSXCell{HasFormula: true}), "unavailable") {
		t.Fatal("missing formula not disclosed")
	}
	tab.tableSelectionKind = "row"
	tab.tableSelectionIndex = 0
	if tableSelectionText(tab, false) != "0.25\t25" || tableSelectionText(tab, true) != "25%\t25" {
		t.Fatal("row selection values")
	}
	tab.tableSelectionKind = "column"
	tab.tableSelectionIndex = 0
	if tableSelectionText(tab, false) != "0.25\ntext" {
		t.Fatal("column selection values")
	}
	if !tableAddressInPage(tab.tablePage, "B2") || tableAddressInPage(tab.tablePage, "C2") {
		t.Fatal("blank cell/range selection")
	}
	for _, pair := range []struct{ r, c int }{{0, 0}, {1, 26}, {60, 16383}} {
		address := tableAddress(pair.r, pair.c)
		r, c, ok := tableCoordinates(address)
		if !ok || r != pair.r || c != pair.c {
			t.Fatal(address)
		}
	}
}
func TestTableGridReadOnlyCellAndHeaderClicks(t *testing.T) {
	m, tab := tableGridFixture()
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { paintTableGrid(c, m) }) }, 700, 400)
	if err := tt.Click("Cell B2"); err != nil {
		t.Fatal(err)
	}
	if tab.tableCell != "B2" || tab.tableSelectionKind != "cell" {
		t.Fatal("blank cell not selectable")
	}
	if err := tt.Click("B"); err != nil {
		t.Fatal(err)
	}
	if tab.tableSelectionKind != "column" || tab.tableSelectionIndex != 1 {
		t.Fatal("column not selectable")
	}
	if err := tt.Click("2"); err != nil {
		t.Fatal(err)
	}
	if tab.tableSelectionKind != "row" || tab.tableSelectionIndex != 1 {
		t.Fatal("row not selectable")
	}
	if err := tt.Click("Read-only cell formula or value"); err != nil {
		t.Fatal(err)
	}
	tt.Type("overwrite")
	if tab.tablePage.Cells[2].Value != "text" || tab.Editing {
		t.Fatal("formula bar changed workbook")
	}
}
