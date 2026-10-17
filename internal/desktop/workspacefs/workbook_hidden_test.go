package workspacefs

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
)

func TestWorkbookSavedHiddenRowsColumns(t *testing.T) {
	var data bytes.Buffer
	z := zip.NewWriter(&data)
	w, _ := z.Create("xl/worksheets/sheet1.xml")
	_, _ = w.Write([]byte(`<worksheet><sheetFormatPr zeroHeight="1"/><cols><col min="2" max="2" hidden="1"/></cols><sheetData><row r="1" hidden="0"><c r="A1"><v>1</v></c><c r="B1"><v>2</v></c></row><row r="2"><c r="A2"><v>3</v></c></row></sheetData></worksheet>`))
	_ = z.Close()
	page, err := ReadWorkbookBytesPage(context.Background(), data.Bytes(), XLSXPageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !page.DefaultRowsHidden || page.HiddenRows[0] || !page.HiddenRows[1] || !page.HiddenColumns[1] {
		t.Fatalf("hidden metadata %+v", page)
	}
	if len(page.Cells) != 3 || page.Cells[1].Value != "2" {
		t.Fatal("hidden state deleted source data")
	}
}
