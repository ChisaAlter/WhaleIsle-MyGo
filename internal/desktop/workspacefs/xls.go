package workspacefs

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	xls "github.com/ronohara/go-xls-reader"
)

func readXLSPage(ctx context.Context, reader io.ReadSeeker, version string, request XLSXPageRequest) (*XLSXPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if request.Row < 0 || request.Row >= 65536 || request.Column < 0 || request.Column >= 256 {
		return nil, errors.New("XLS page is outside worksheet coordinates")
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
	book, err := xls.OpenReader(xlsContextReadSeeker{ctx: ctx, reader: reader})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if book == nil {
		return nil, errors.New("XLS does not contain a BIFF workbook stream")
	}
	out := &XLSXPage{Request: request, Version: version, TotalRows: 1, TotalColumns: 1}
	selected, sheetName := -1, ""
	for _, sheet := range book.Worksheets() {
		id := "xls:" + strconv.Itoa(sheet.Index)
		out.Sheets = append(out.Sheets, XLSXSheet{ID: id, Name: sheet.Name})
		if request.Sheet == "" {
			request.Sheet = id
		}
		if request.Sheet == id {
			selected, sheetName = sheet.Index, sheet.Name
		}
	}
	if selected < 0 {
		return nil, errors.New("Selected worksheet no longer exists; choose Reload")
	}
	sheet := book.GetSheet(selected)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out.TotalRows = max(1, int(sheet.MaxRow)+1)
	out.TotalColumns = max(1, sheet.MaxCol()+1)
	request.Version = version
	out.Request = request
	rows := max(0, min(request.Rows, out.TotalRows-request.Row))
	columns := max(0, min(request.Columns, out.TotalColumns-request.Column))
	grid := make([][]string, rows)
	displayGrid := make([][]string, rows)
	pageBytes := 0
	for row := 0; row < rows; row++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		grid[row] = make([]string, columns)
		displayGrid[row] = make([]string, columns)
		for col := 0; col < columns; col++ {
			cell := book.TypedCell(selected, request.Row+row, request.Column+col)
			if !cell.Present || cell.Kind == "blank" {
				continue
			}
			value := xlsStoredValue(cell)
			address := xlsxColumnName(request.Column+col+1) + strconv.Itoa(request.Row+row+1)
			format := book.FormatText(cell.FormatNo)
			if format == "" {
				format = workbookBuiltinFormat(int(cell.FormatNo))
			}
			kind := cell.Kind
			if kind == "bool" {
				kind = "boolean"
			}
			display := workbookCell(address, value, kind, format, book.DateMode() == 1)
			if max(len(value), len(display.DisplayValue)) > xlsxMaxCellBytes && out.Problem == "" {
				out.Problem = fmt.Sprintf("Cell %s exceeds the 64 KiB readable cell limit. This page is not shown or copyable. Choose another page or Open external.", address)
			}
			pageBytes += max(len(value), len(display.DisplayValue))
			if pageBytes > xlsxMaxPageBytes && out.Problem == "" {
				out.Problem = "Selected page exceeds the 1 MiB cell-content limit. This page is not shown or copyable. Choose another page or Open external."
			}
			if out.Problem != "" {
				continue
			}
			grid[row][col] = value
			displayGrid[row][col] = display.DisplayValue
			out.Cells = append(out.Cells, display)
		}
	}
	if out.Problem != "" {
		out.Cells = nil
	} else {
		for index, values := range [][][]string{grid, displayGrid} {
			var body strings.Builder
			writer := csv.NewWriter(&body)
			writer.Comma = '\t'
			for _, row := range values {
				if err := writer.Write(row); err != nil {
					return nil, err
				}
			}
			writer.Flush()
			if err := writer.Error(); err != nil {
				return nil, err
			}
			if index == 0 {
				out.TSV = body.String()
			} else {
				out.DisplayTSV = body.String()
			}
		}
	}
	var text strings.Builder
	fmt.Fprintf(&text, "%s · rows %d–%d / %d · columns %s–%s / %s\nRead-only stored values with supported number formats; formulas are not recalculated. Raw and displayed values can be copied separately. Use worksheet and row/column controls for other pages.\n\n", sheetName, request.Row+1, min(request.Row+request.Rows, out.TotalRows), out.TotalRows, xlsxColumnName(request.Column+1), xlsxColumnName(min(request.Column+request.Columns, out.TotalColumns)), xlsxColumnName(out.TotalColumns))
	escape := strings.NewReplacer("\\", "\\\\", "\r", "\\r", "\n", "\\n", "\t", "\\t")
	for _, cell := range out.Cells {
		text.WriteString(cell.Address + "\t" + escape.Replace(cell.DisplayValue) + "\n")
	}
	if out.Problem != "" {
		text.WriteString(out.Problem)
	} else if len(out.Cells) == 0 {
		text.WriteString("(No stored cells in this page.)")
	}
	text.WriteString(workbookFormatNotice(out.Cells))
	out.Text = text.String()
	return out, nil
}

func xlsStoredValue(cell xls.Cell) string {
	switch cell.Kind {
	case "number":
		return strconv.FormatFloat(cell.Number, 'f', -1, 64)
	case "bool":
		if cell.Bool {
			return "TRUE"
		}
		return "FALSE"
	case "error":
		return xls.ErrorText(cell.Error)
	default:
		return cell.Text
	}
}

type xlsContextReadSeeker struct {
	ctx    context.Context
	reader io.ReadSeeker
}

func (r xlsContextReadSeeker) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func (r xlsContextReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Seek(offset, whence)
}
