package workspacefs

import (
	"encoding/csv"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// IsCSVPath reports whether path looks like a CSV/TSV preview candidate.
func IsCSVPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv", ".tsv":
		return true
	default:
		return false
	}
}

// FormatCSVTable renders a delimited text blob as a padded monospace table.
// Returns ok=false when the content is not a usable CSV/TSV grid.
func FormatCSVTable(text string, maxRows, maxCols int) (string, bool) {
	text = strings.TrimPrefix(text, "\ufeff")
	if strings.TrimSpace(text) == "" {
		return "", false
	}
	if maxRows < 1 {
		maxRows = 50
	}
	if maxCols < 1 {
		maxCols = 12
	}
	delim := ','
	if strings.Count(text, "\t") > strings.Count(text, ",") {
		delim = '\t'
	}
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil || len(records) == 0 {
		return "", false
	}
	if len(records) == 1 && len(records[0]) < 2 {
		return "", false
	}
	truncated := false
	if len(records) > maxRows {
		records = records[:maxRows]
		truncated = true
	}
	widths := make([]int, 0)
	for _, row := range records {
		for i, cell := range row {
			if i >= maxCols {
				break
			}
			for len(widths) <= i {
				widths = append(widths, 0)
			}
			n := utf8.RuneCountInString(cell)
			if n > 40 {
				n = 40
			}
			if n > widths[i] {
				widths[i] = n
			}
		}
	}
	if len(widths) == 0 {
		return "", false
	}
	var b strings.Builder
	for ri, row := range records {
		if ri > 0 {
			b.WriteByte('\n')
		}
		for i, w := range widths {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			runes := []rune(cell)
			if len(runes) > 40 {
				cell = string(runes[:39]) + "…"
			}
			b.WriteString(padRight(cell, w))
			if i+1 < len(widths) {
				b.WriteString("  ")
			}
		}
		if len(row) > maxCols {
			b.WriteString("  …")
		}
	}
	if truncated {
		b.WriteString("\n…")
	}
	return b.String(), true
}

func padRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}
