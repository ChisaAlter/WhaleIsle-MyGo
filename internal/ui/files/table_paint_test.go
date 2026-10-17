package files

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestTableSourceFillBordersReadOnly(t *testing.T) {
	m, tab := tableGridFixture()
	tab.tableCell = "B2"
	tab.tablePage.Cells[0].Style = workspacefs.XLSXCellStyle{FillPattern: "solid", FillColor: "#AABBCC", Borders: [4]workspacefs.XLSXBorder{{Style: "dashed", Color: "#112233"}, {Style: "double", Color: "#112233"}}}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { paintTableGrid(c, m) }) }, 700, 400)
	foundFill, foundBorder := false, false
	bitmap := tt.Image()
	for y := 0; y < bitmap.Bounds().Dy(); y++ {
		for x := 0; x < bitmap.Bounds().Dx(); x++ {
			color := bitmap.RGBAAt(x, y)
			if color.R == 0xAA && color.G == 0xBB && color.B == 0xCC {
				foundFill = true
			}
			if color.R == 0x11 && color.G == 0x22 && color.B == 0x33 {
				foundBorder = true
			}
		}
	}
	if !foundFill || !foundBorder {
		t.Fatal("source paint absent", foundFill, foundBorder)
	}
	if err := tt.Click("Cell A1"); err != nil {
		t.Fatal(err)
	}
	if tab.tableCell != "A1" || tab.tablePage.Cells[0].Value != "0.25" {
		t.Fatal("source paint changed selection or value")
	}
	if _, ok := tablePaintColor("invalid"); ok {
		t.Fatal("invalid color accepted")
	}
	if !tablePaintUnsupported(workspacefs.XLSXCellStyle{PaintUnsupported: true}) {
		t.Fatal("unsupported source paint hidden")
	}
}
