package workspacefs

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func readDelimitedPage(ctx context.Context, data []byte, comma rune, request XLSXPageRequest) (*XLSXPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(data)) > xlsxMaxFileBytes {
		return nil, errors.New("Table exceeds the 16 MiB native reader limit")
	}
	version := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	if request.Version != "" && request.Version != version {
		return nil, errors.New("Workbook changed; choose Reload before reading another page")
	}
	if request.Row < 0 || request.Column < 0 {
		return nil, errors.New("Table page is outside worksheet coordinates")
	}
	if request.Sheet != "" && request.Sheet != "table" {
		return nil, errors.New("Selected worksheet no longer exists; choose Reload")
	}
	if request.Rows < 1 || request.Rows > xlsxMaxRows {
		request.Rows = xlsxMaxRows
	}
	if request.Columns < 1 || request.Columns > xlsxMaxCols {
		request.Columns = xlsxMaxCols
	}
	request.Sheet, request.Version = "table", version
	out := &XLSXPage{Request: request, Version: version, Sheets: []XLSXSheet{{ID: "table", Name: "Table"}}}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	reader.Comma, reader.FieldsPerRecord = comma, -1
	var grid [][]string
	pageBytes := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		row := out.TotalRows
		out.TotalRows++
		out.TotalColumns = max(out.TotalColumns, len(record))
		if row < request.Row || row-request.Row >= request.Rows {
			continue
		}
		values := make([]string, request.Columns)
		for col, value := range record {
			if col < request.Column || col-request.Column >= request.Columns {
				continue
			}
			pageBytes += len(value)
			if len(value) > xlsxMaxCellBytes || pageBytes > xlsxMaxPageBytes {
				out.Problem = "Selected page exceeds the cell or page content limit. Choose another page or Open external."
				continue
			}
			values[col-request.Column] = value
			out.Cells = append(out.Cells, XLSXCell{Address: xlsxColumnName(col+1) + strconv.Itoa(row+1), Value: value, DisplayValue: value, Type: "string", FormatSupported: true})
		}
		grid = append(grid, values)
	}
	if out.Problem != "" {
		out.Cells = nil
		out.Text = out.Problem
		return out, nil
	}
	columns := max(0, min(request.Columns, out.TotalColumns-request.Column))
	var copied strings.Builder
	writer := csv.NewWriter(&copied)
	writer.Comma = '\t'
	for _, row := range grid {
		if err := writer.Write(row[:columns]); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	out.TSV, out.DisplayTSV = copied.String(), copied.String()
	var text strings.Builder
	fmt.Fprintf(&text, "Table · %d rows × %d columns\nRead-only text values; formulas are not evaluated.\n\n", out.TotalRows, out.TotalColumns)
	escape := strings.NewReplacer("\\", "\\\\", "\r", "\\r", "\n", "\\n", "\t", "\\t")
	for _, cell := range out.Cells {
		fmt.Fprintf(&text, "%s\t%s\n", cell.Address, escape.Replace(cell.Value))
	}
	out.Text = text.String()
	return out, nil
}
