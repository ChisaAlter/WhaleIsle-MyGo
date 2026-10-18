package workspacefs

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
)

func TestWorkbookFormulaAndSavedPresentation(t *testing.T) {
	var data bytes.Buffer
	z := zip.NewWriter(&data)
	parts := map[string]string{
		"xl/styles.xml":            `<styleSheet><fonts><font><name val="Calibri"/><sz val="11"/></font><font><name val="Arial"/><sz val="14"/><b/><i val="1"/></font></fonts><cellXfs><xf numFmtId="0" fontId="0"/><xf numFmtId="10" fontId="1"><alignment horizontal="right" vertical="top" wrapText="1"/></xf></cellXfs></styleSheet>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData><row r="1"><c r="A1" s="1"><f>SUM(B1:C1)</f><v>0.25</v></c><c r="B1"><f t="shared" si="0"/><v>2</v></c></row></sheetData></worksheet>`}
	for name, body := range parts {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	page, err := ReadWorkbookBytesPage(context.Background(), data.Bytes(), XLSXPageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	cell := page.Cells[0]
	if cell.Formula != "SUM(B1:C1)" || !cell.HasFormula || cell.Value != "0.25" || cell.DisplayValue != "25.00%" {
		t.Fatalf("formula/stored value: %+v", cell)
	}
	if !cell.Style.Available || cell.Style.FontName != "Arial" || cell.Style.FontSize != 14 || !cell.Style.Bold || !cell.Style.Italic || cell.Style.Horizontal != "right" || !cell.Style.Wrap {
		t.Fatalf("style %+v", cell.Style)
	}
	if !page.Cells[1].HasFormula || page.Cells[1].Formula != "" || page.Cells[1].FormulaKind != "shared" {
		t.Fatal("shared formula absence was hidden")
	}
}
