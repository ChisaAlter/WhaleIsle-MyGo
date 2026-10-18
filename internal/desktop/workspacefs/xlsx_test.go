package workspacefs

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatXLSXPreview(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.xlsx")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("[Content_Types].xml")
	_, _ = w.Write([]byte(`<?xml version="1.0"?><Types></Types>`))
	w, _ = zw.Create("xl/sharedStrings.xml")
	_, _ = w.Write([]byte(`<?xml version="1.0"?><sst><si><t>hello</t></si><si><t>world</t></si></sst>`))
	w, _ = zw.Create("xl/worksheets/sheet1.xml")
	_, _ = w.Write([]byte(`<?xml version="1.0"?><worksheet><sheetData>
<row><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
<row><c r="A2"><v>3</v></c><c r="B2"><v>4</v></c></row>
</sheetData></worksheet>`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	text, err := FormatXLSXPreview(path, 20, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "hello") || !strings.Contains(text, "world") || !strings.Contains(text, "3") {
		t.Fatalf("%q", text)
	}
}
