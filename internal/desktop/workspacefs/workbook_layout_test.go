package workspacefs

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
)

func TestWorkbookSavedLayoutAndInheritance(t *testing.T) {
	var data bytes.Buffer
	z := zip.NewWriter(&data)
	parts := map[string]string{
		"xl/styles.xml":            `<styleSheet><fonts><font/><font><name val="Arial"/><sz val="12"/><b/></font></fonts><cellStyleXfs><xf fontId="1" numFmtId="10"><alignment wrapText="1" horizontal="right"/></xf></cellStyleXfs><cellXfs><xf xfId="0"/><xf xfId="0" fontId="0" numFmtId="0" applyFont="0" applyNumberFormat="0"/></cellXfs></styleSheet>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetFormatPr defaultRowHeight="15" defaultColWidth="9"/><cols><col min="1" max="2" width="22"/></cols><sheetData><row r="1" ht="30"><c r="A1"><v>0.25</v></c></row><row r="2"><c r="B2" s="1"><v>0.5</v></c></row></sheetData><mergeCells><mergeCell ref="A1:B2"/></mergeCells></worksheet>`,
	}
	for name, body := range parts {
		w, _ := z.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = z.Close()
	page, err := ReadWorkbookBytesPage(context.Background(), data.Bytes(), XLSXPageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Merges) != 1 || page.Merges[0].LastColumn != 1 || page.Merges[0].LastRow != 1 {
		t.Fatalf("merges %+v", page.Merges)
	}
	if page.RowHeights[0] != 30 || page.ColumnWidths[1] != 22 || page.DefaultRowHeight != 15 || page.DefaultColumnWidth != 9 {
		t.Fatalf("layout %+v", page)
	}
	for _, cell := range page.Cells {
		if cell.Style.FontName != "Arial" || !cell.Style.Wrap || cell.DisplayValue == cell.Value {
			t.Fatalf("inheritance %+v", cell)
		}
	}
}
