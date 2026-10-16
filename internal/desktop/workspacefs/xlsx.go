package workspacefs

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	xlsxMaxFileBytes     int64  = 16 << 20
	xlsxMaxXMLBytes      uint64 = 8 << 20
	xlsxMaxTotalXMLBytes uint64 = 32 << 20
	xlsxMaxEntries              = 4096
	xlsxMaxSharedStrings        = 200000
	xlsxMaxRows                 = 60
	xlsxMaxCols                 = 16
	xlsxMaxCellBytes            = 64 << 10
	xlsxMaxPageBytes            = 1 << 20
)

// XLSXPageRequest addresses a bounded viewport in an already-authorized workbook.
// Sheet is an internal worksheet part ID returned by ReadXLSXPage, never a path
// to open on disk. Version binds subsequent page reads to the same local file.
type XLSXPageRequest struct {
	Sheet         string
	Row, Column   int // zero-based
	Rows, Columns int
	Version       string
}
type XLSXSheet struct{ ID, Name string }
type XLSXCell struct {
	Address, Value, Type, DisplayValue, NumberFormat string
	FormatSupported                                bool
}
type XLSXPage struct {
	Request                                XLSXPageRequest
	Sheets                                 []XLSXSheet
	Cells                                  []XLSXCell
	TotalRows, TotalColumns                 int
	Text, TSV, DisplayTSV, Version, Problem string
}

// FormatXLSXPreview keeps the original API for standalone callers. The UI uses
// ReadXLSXPage and exposes sheet/row/column navigation and exact cell copies.
func FormatXLSXPreview(filePath string, maxRows, maxCols int) (string, error) {
	page, err := ReadXLSXPage(context.Background(), filePath, XLSXPageRequest{Rows: maxRows, Columns: maxCols})
	if err != nil {
		return "", err
	}
	return page.Text, nil
}

// ReadXLSXPage reads stored values, never formulas/macros/external relationships.
// Authorize filePath with JoinRoot first. Every page rereads only this file.
func ReadXLSXPage(ctx context.Context, filePath string, request XLSXPageRequest) (*XLSXPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("XLSX preview requires a regular file")
	}
	if st.Size() > xlsxMaxFileBytes {
		return nil, errors.New("XLSX exceeds the 16 MiB native reader limit; use Open external")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	st, err = file.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > xlsxMaxFileBytes {
		return nil, errors.New("XLSX changed or exceeds the 16 MiB native reader limit")
	}
	version := fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixNano())
	page, err := readXLSXPage(ctx, file, st.Size(), version, request)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	current, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(st, current) || after.Size() != st.Size() || !after.ModTime().Equal(st.ModTime()) {
		return nil, errors.New("Workbook changed during reading; choose Reload")
	}
	return page, nil
}

// ReadXLSXBytesPage uses exactly the same workbook/sheet/page reader as local files.
func ReadXLSXBytesPage(ctx context.Context, data []byte, request XLSXPageRequest) (*XLSXPage, error) {
	if int64(len(data)) > xlsxMaxFileBytes {
		return nil, errors.New("XLSX exceeds the 16 MiB native reader limit")
	}
	version := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	return readXLSXPage(ctx, bytes.NewReader(data), int64(len(data)), version, request)
}
func readXLSXPage(ctx context.Context, reader io.ReaderAt, size int64, version string, request XLSXPageRequest) (*XLSXPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if request.Row < 0 || request.Row >= 1048576 || request.Column < 0 || request.Column >= 16384 {
		return nil, errors.New("XLSX page is outside worksheet coordinates")
	}
	if request.Rows < 1 || request.Rows > xlsxMaxRows {
		request.Rows = xlsxMaxRows
	}
	if request.Columns < 1 || request.Columns > xlsxMaxCols {
		request.Columns = xlsxMaxCols
	}
	if request.Version != "" && request.Version != version {
		return nil, errors.New("Workbook changed; choose Reload before reading another page")
	}
	zr, err := zip.NewReader(xlsxContextReaderAt{ctx, reader}, size)
	if err != nil {
		return nil, err
	}
	if len(zr.File) > xlsxMaxEntries {
		return nil, errors.New("XLSX exceeds the archive entry limit (4096)")
	}
	var xmlBytes uint64
	seen := map[string]bool{}
	for _, f := range zr.File {
		if len(f.Name) > 1024 || seen[f.Name] {
			return nil, errors.New("XLSX has an invalid or duplicate archive part")
		}
		seen[f.Name] = true
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			if f.UncompressedSize64 > xlsxMaxXMLBytes || xmlBytes > xlsxMaxTotalXMLBytes-f.UncompressedSize64 {
				return nil, errors.New("XLSX exceeds the XML limit (8 MiB per part, 32 MiB total); use Open external")
			}
			xmlBytes += f.UncompressedSize64
		}
	}
	sheets, err := xlsxSheets(ctx, zr, seen)
	if err != nil {
		return nil, err
	}
	if request.Sheet == "" {
		request.Sheet = sheets[0].ID
	}
	sheetName := ""
	for _, sheet := range sheets {
		if sheet.ID == request.Sheet {
			sheetName = sheet.Name
			break
		}
	}
	if sheetName == "" {
		return nil, errors.New("Selected worksheet no longer exists; choose Reload")
	}
	shared, err := readSharedStrings(ctx, zr)
	if err != nil {
		return nil, err
	}
	formats, err := readWorkbookFormats(ctx, zr, seen)
	if err != nil {
		return nil, err
	}
	page, err := readSheetPage(ctx, zr, request, shared, formats)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	page.Sheets, page.Version = sheets, version
	page.Request.Version = version
	page.Text = fmt.Sprintf("%s · rows %d–%d / %d · columns %s–%s / %s\nRead-only stored values with supported number formats; formulas are not recalculated. Raw and displayed values can be copied separately. Use worksheet and row/column controls for other pages.\n\n", sheetName, request.Row+1, min(request.Row+request.Rows, page.TotalRows), page.TotalRows, xlsxColumnName(request.Column+1), xlsxColumnName(min(request.Column+request.Columns, page.TotalColumns)), xlsxColumnName(page.TotalColumns))
	var body strings.Builder
	body.WriteString(page.Text)
	escape := strings.NewReplacer("\\", "\\\\", "\r", "\\r", "\n", "\\n", "\t", "\\t")
	for _, cell := range page.Cells {
		body.WriteString(cell.Address + "\t" + escape.Replace(cell.DisplayValue) + "\n")
	}
	if page.Problem != "" {
		body.WriteString(page.Problem)
	} else if len(page.Cells) == 0 {
		body.WriteString("(No stored cells in this page.)")
	}
	body.WriteString(workbookFormatNotice(page.Cells))
	page.Text = body.String()
	return page, nil
}

func xlsxSheets(ctx context.Context, zr *zip.Reader, seen map[string]bool) ([]XLSXSheet, error) {
	var sheets []XLSXSheet
	if !seen["xl/workbook.xml"] && !seen["xl/_rels/workbook.xml.rels"] {
		// Compatibility with simple value-only archives; do not imply workbook order.
		for _, f := range zr.File {
			if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
				sheets = append(sheets, XLSXSheet{f.Name, path.Base(f.Name) + " (archive order)"})
			}
		}
	} else {
		rc, dec, err := xlsxPart(ctx, zr, "xl/workbook.xml")
		if err != nil {
			return nil, err
		}
		var book struct {
			Sheets []struct {
				Name string `xml:"name,attr"`
				ID   string `xml:"id,attr"`
			} `xml:"sheets>sheet"`
		}
		err = dec.Decode(&book)
		rc.Close()
		if err != nil {
			return nil, err
		}
		rc, dec, err = xlsxPart(ctx, zr, "xl/_rels/workbook.xml.rels")
		if err != nil {
			return nil, err
		}
		var rels struct {
			Items []struct {
				ID     string `xml:"Id,attr"`
				Target string `xml:"Target,attr"`
				Mode   string `xml:"TargetMode,attr"`
				Type   string `xml:"Type,attr"`
			} `xml:"Relationship"`
		}
		err = dec.Decode(&rels)
		rc.Close()
		if err != nil {
			return nil, err
		}
		targets := map[string]string{}
		for _, rel := range rels.Items {
			if rel.Mode != "" && rel.Mode != "Internal" {
				continue
			}
			if !strings.HasSuffix(rel.Type, "/worksheet") {
				continue
			}
			target := path.Clean(path.Join("xl", rel.Target))
			if strings.HasPrefix(rel.Target, "/") {
				target = path.Clean(strings.TrimPrefix(rel.Target, "/"))
			}
			if !strings.HasPrefix(target, "xl/worksheets/") || !seen[target] {
				continue
			}
			targets[rel.ID] = target
		}
		for _, sheet := range book.Sheets {
			target, ok := targets[sheet.ID]
			if !ok || sheet.Name == "" || len(sheet.Name) > 1024 {
				return nil, errors.New("XLSX has an unsupported or invalid worksheet relationship")
			}
			sheets = append(sheets, XLSXSheet{target, sheet.Name})
		}
	}
	if len(sheets) == 0 {
		return nil, errors.New("no worksheet")
	}
	return sheets, nil
}

type xlsxContextReaderAt struct {
	ctx    context.Context
	reader io.ReaderAt
}

func (r xlsxContextReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.ReadAt(p, off)
}

type xlsxContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r xlsxContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func xlsxPart(ctx context.Context, zr *zip.Reader, name string) (io.ReadCloser, *xml.Decoder, error) {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		if f.UncompressedSize64 > xlsxMaxXMLBytes {
			return nil, nil, errors.New("XLSX XML part exceeds limit")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		return rc, xml.NewDecoder(xlsxContextReader{ctx, io.LimitReader(rc, int64(xlsxMaxXMLBytes)+1)}), nil
	}
	return nil, nil, fmt.Errorf("missing %s", name)
}

type xlsxRichText struct {
	T string `xml:"t"`
	R []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

func (v xlsxRichText) text() string {
	if len(v.R) == 0 {
		return v.T
	}
	var b strings.Builder
	b.WriteString(v.T)
	for _, run := range v.R {
		b.WriteString(run.T)
	}
	return b.String()
}
func readSharedStrings(ctx context.Context, zr *zip.Reader) ([]string, error) {
	exists := false
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			exists = true
			break
		}
	}
	if !exists {
		return nil, nil
	}
	rc, dec, err := xlsxPart(ctx, zr, "xl/sharedStrings.xml")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var out []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "si" {
			continue
		}
		if len(out) >= xlsxMaxSharedStrings {
			return nil, errors.New("XLSX exceeds the shared string limit")
		}
		var item xlsxRichText
		if err := dec.DecodeElement(&item, &start); err != nil {
			return nil, err
		}
		out = append(out, item.text())
	}
}
func readSheetPage(ctx context.Context, zr *zip.Reader, request XLSXPageRequest, shared []string, formats workbookFormats) (*XLSXPage, error) {
	rc, dec, err := xlsxPart(ctx, zr, request.Sheet)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	type cell struct {
		R      string       `xml:"r,attr"`
		T      string       `xml:"t,attr"`
		S      *int         `xml:"s,attr"`
		V      string       `xml:"v"`
		Inline xlsxRichText `xml:"is"`
	}
	type row struct {
		R int    `xml:"r,attr"`
		S *int   `xml:"s,attr"`
		C []cell `xml:"c"`
	}
	type columnFormat struct {
		Min   int `xml:"min,attr"`
		Max   int `xml:"max,attr"`
		Style int `xml:"style,attr"`
	}
	var columnFormats []columnFormat
	out := &XLSXPage{Request: request, TotalRows: 1, TotalColumns: 1}
	grid := make([][]string, request.Rows)
	displayGrid := make([][]string, request.Rows)
	for i := range grid {
		grid[i] = make([]string, request.Columns)
		displayGrid[i] = make([]string, request.Columns)
	}
	lastRow, pageBytes := 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "col" {
			var column columnFormat
			if err := dec.DecodeElement(&column, &start); err != nil {
				return nil, err
			}
			columnFormats = append(columnFormats, column)
			continue
		}
		if start.Name.Local != "row" {
			continue
		}
		var r row
		if err := dec.DecodeElement(&r, &start); err != nil {
			return nil, err
		}
		rowNum := r.R
		if rowNum == 0 {
			rowNum = lastRow + 1
		}
		if rowNum <= lastRow || rowNum > 1048576 {
			return nil, errors.New("XLSX has invalid row coordinates")
		}
		lastRow = rowNum
		out.TotalRows = max(out.TotalRows, rowNum)
		lastCol := -1
		for _, c := range r.C {
			col := colIndex(c.R)
			if c.R == "" {
				col = lastCol + 1
			}
			if col < 0 || col >= 16384 || col <= lastCol {
				return nil, errors.New("XLSX has invalid cell coordinates")
			}
			if c.R != "" {
				suffix := strings.TrimLeft(c.R, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
				cellRow, e := strconv.Atoi(suffix)
				if e != nil || cellRow != rowNum {
					return nil, errors.New("XLSX cell does not match its row")
				}
			}
			lastCol = col
			out.TotalColumns = max(out.TotalColumns, col+1)
			if rowNum-1 < request.Row || rowNum-1 >= request.Row+request.Rows || col < request.Column || col >= request.Column+request.Columns {
				continue
			}
			val := c.V
			if c.T == "inlineStr" {
				val = c.Inline.text()
			}
			if c.T == "s" {
				idx, err := strconv.Atoi(val)
				if err != nil || idx < 0 || idx >= len(shared) {
					return nil, errors.New("XLSX contains an invalid shared string reference")
				}
				val = shared[idx]
			}
			address := xlsxColumnName(col+1) + strconv.Itoa(rowNum)
			kind := "number"
			switch c.T {
			case "inlineStr", "s", "str":
				kind = "text"
			case "b":
				kind = "boolean"
			case "e":
				kind = "error"
			case "d":
				kind = "date"
			}
			style := 0
			for _, column := range columnFormats {
				if col+1 >= column.Min && col+1 <= column.Max {
					style = column.Style
				}
			}
			if r.S != nil {
				style = *r.S
			}
			if c.S != nil {
				style = *c.S
			}
			display := workbookCell(address, val, kind, formats.code(style), formats.date1904)
			if max(len(val), len(display.DisplayValue)) > xlsxMaxCellBytes && out.Problem == "" {
				out.Problem = fmt.Sprintf("Cell %s exceeds the 64 KiB readable cell limit. This page is not shown or copyable. Choose another page or Open external.", address)
			}
			pageBytes += max(len(val), len(display.DisplayValue))
			if pageBytes > xlsxMaxPageBytes && out.Problem == "" {
				out.Problem = "Selected page exceeds the 1 MiB cell-content limit. This page is not shown or copyable. Choose another page or Open external."
			}
			if out.Problem != "" {
				continue
			}
			grid[rowNum-1-request.Row][col-request.Column] = val
			displayGrid[rowNum-1-request.Row][col-request.Column] = display.DisplayValue
			out.Cells = append(out.Cells, display)
		}
	}
	if out.Problem != "" {
		out.Cells = nil
		return out, nil
	}
	for index, values := range [][][]string{grid, displayGrid} {
		var b strings.Builder
		writer := csv.NewWriter(&b)
		writer.Comma = '\t'
		for _, row := range values[:max(0, min(request.Rows, out.TotalRows-request.Row))] {
			if err := writer.Write(row[:max(0, min(request.Columns, out.TotalColumns-request.Column))]); err != nil {
				return nil, err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, err
		}
		if index == 0 {
			out.TSV = b.String()
		} else {
			out.DisplayTSV = b.String()
		}
	}
	return out, nil
}
func xlsxColumnName(n int) string {
	if n < 1 {
		return ""
	}
	var name string
	for n > 0 {
		n--
		name = string(rune('A'+n%26)) + name
		n /= 26
	}
	return name
}
func colIndex(ref string) int {
	n := 0
	for _, ch := range ref {
		if ch < 'A' || ch > 'Z' {
			break
		}
		// XLSX has at most 16384 columns; guard malicious/overflowing references.
		if n > 16384 {
			return -1
		}
		n = n*26 + int(ch-'A'+1)
	}
	if n > 16384 {
		return -1
	}
	return n - 1
}
