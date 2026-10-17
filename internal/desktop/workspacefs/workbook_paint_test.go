package workspacefs

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
)

func TestWorkbookSavedFillBorderInheritance(t *testing.T) {
	var data bytes.Buffer
	z := zip.NewWriter(&data)
	parts := map[string]string{
		"xl/styles.xml":            `<styleSheet><fonts><font/></fonts><fills><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="solid"><fgColor theme="4"/></patternFill></fill></fills><borders><border/><border><left style="double"><color rgb="FF112233"/></left><bottom style="dashed"><color indexed="0"/></bottom></border></borders><colors><indexedColors><rgbColor rgb="FF445566"/></indexedColors></colors><cellStyleXfs><xf fillId="1" borderId="1"/></cellStyleXfs><cellXfs><xf xfId="0"/></cellXfs></styleSheet>`,
		"xl/theme/theme1.xml":      `<a:theme xmlns:a="urn:test"><a:themeElements><a:clrScheme><a:accent1><a:srgbClr val="AABBCC"/></a:accent1></a:clrScheme></a:themeElements></a:theme>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData><row r="1"><c r="A1"><v>7</v></c></row></sheetData></worksheet>`,
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
	style := page.Cells[0].Style
	if style.FillPattern != "solid" || style.FillColor != "#AABBCC" || style.Borders[3].Style != "double" || style.Borders[3].Color != "#112233" || style.Borders[2].Color != "#445566" || style.PaintUnsupported {
		t.Fatalf("paint %+v", style)
	}
	if _, ok := savedXLSXColor(xlsxSourceColor{RGB: "not hex"}, nil, nil); ok {
		t.Fatal("invented invalid color")
	}
	if _, ok := savedXLSXColor(xlsxSourceColor{RGB: "FF112233", Tint: .4}, nil, nil); ok {
		t.Fatal("silently ignored unsupported tint")
	}
}
