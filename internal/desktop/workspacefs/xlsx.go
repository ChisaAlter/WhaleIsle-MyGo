package workspacefs

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FormatXLSXPreview extracts a plain-text table from the first worksheet of an
// .xlsx (Office Open XML). Limited to shared strings + inline values; no formulas.
func FormatXLSXPreview(path string, maxRows, maxCols int) (string, error) {
	if maxRows < 1 {
		maxRows = 40
	}
	if maxCols < 1 {
		maxCols = 12
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	shared, err := readSharedStrings(zr)
	if err != nil {
		return "", err
	}
	sheetName := ""
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheetName = f.Name
			break
		}
	}
	if sheetName == "" {
		return "", fmt.Errorf("no worksheet")
	}
	rows, err := readSheetRows(zr, sheetName, shared, maxRows, maxCols)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("empty sheet")
	}
	// Reuse CSV table formatter for alignment.
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.Join(row, ","))
	}
	table, ok := FormatCSVTable(b.String(), maxRows, maxCols)
	if !ok {
		return b.String(), nil
	}
	return table, nil
}

func readSharedStrings(zr *zip.ReadCloser) ([]string, error) {
	var f *zip.File
	for _, cand := range zr.File {
		if cand.Name == "xl/sharedStrings.xml" {
			f = cand
			break
		}
	}
	if f == nil {
		return nil, nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	type tNode struct {
		T string `xml:"t"`
	}
	type si struct {
		T  string  `xml:"t"`
		R  []tNode `xml:"r"`
	}
	type sst struct {
		SI []si `xml:"si"`
	}
	var doc sst
	dec := xml.NewDecoder(rc)
	if err := dec.Decode(&doc); err != nil && err != io.EOF {
		return nil, err
	}
	out := make([]string, 0, len(doc.SI))
	for _, item := range doc.SI {
		if item.T != "" {
			out = append(out, item.T)
			continue
		}
		var b strings.Builder
		for _, r := range item.R {
			b.WriteString(r.T)
		}
		out = append(out, b.String())
	}
	return out, nil
}

func readSheetRows(zr *zip.ReadCloser, name string, shared []string, maxRows, maxCols int) ([][]string, error) {
	var f *zip.File
	for _, cand := range zr.File {
		if cand.Name == name {
			f = cand
			break
		}
	}
	if f == nil {
		return nil, fmt.Errorf("missing %s", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	type c struct {
		R string `xml:"r,attr"`
		T string `xml:"t,attr"`
		V string `xml:"v"`
	}
	type row struct {
		C []c `xml:"c"`
	}
	dec := xml.NewDecoder(rc)
	var out [][]string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "row" {
			continue
		}
		var r row
		if err := dec.DecodeElement(&r, &se); err != nil {
			return nil, err
		}
		cells := make([]string, maxCols)
		used := 0
		for _, cell := range r.C {
			col := colIndex(cell.R)
			if col < 0 || col >= maxCols {
				continue
			}
			val := cell.V
			if cell.T == "s" {
				idx, _ := strconv.Atoi(val)
				if idx >= 0 && idx < len(shared) {
					val = shared[idx]
				}
			}
			cells[col] = val
			if col+1 > used {
				used = col + 1
			}
		}
		out = append(out, cells[:used])
		if len(out) >= maxRows {
			break
		}
	}
	return out, nil
}

func colIndex(ref string) int {
	n := 0
	for _, ch := range ref {
		if ch < 'A' || ch > 'Z' {
			break
		}
		n = n*26 + int(ch-'A'+1)
	}
	return n - 1
}
