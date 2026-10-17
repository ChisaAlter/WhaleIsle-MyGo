package files

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestTableHiddenDimensionsAndMergeOrigin(t *testing.T) {
	m, tab := tableGridFixture()
	page := tab.tablePage
	page.HiddenRows = map[int]bool{0: true}
	page.HiddenColumns = map[int]bool{0: true}
	page.Merges = []workspacefs.XLSXMerge{{FirstRow: 0, LastRow: 1, FirstColumn: 0, LastColumn: 1}}
	if tableRowHeight(page, 0) != 0 || tableColumnWidth(page, 0) != 0 {
		t.Fatal("hidden dimensions occupy space")
	}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { paintTableGrid(c, m) }) }, 700, 400)
	if _, ok := tt.Find("A"); ok {
		t.Fatal("hidden column header still rendered")
	}
	if _, ok := tt.Find("1"); ok {
		t.Fatal("hidden row header still rendered")
	}
	if err := tt.Click("Cell B2"); err != nil {
		t.Fatal(err)
	}
	if tab.tableCell != "A1" || tableSelectionText(tab, false) != "0.25" {
		t.Fatal("visible merge lost hidden origin")
	}
}
