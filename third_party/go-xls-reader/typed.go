package xls

import (
	"regexp"
	"strconv"
	"strings"
)

// typed.go — a typed cell path for this fork.
//
// Upstream computes a cell's value only inside contentHandler.String and keeps
// the XF/format behind unexported fields, so a caller can neither obtain a typed
// value nor the format index that decides whether a number is a date. Inside the
// package both are reachable. This adds that path and applies xlrd's date rule
// (ported from xlrd/formatting.py) rather than upstream's own heuristic, which
// differs from xlrd's on the built-in ranges and on how a custom format string
// is read.

// DateMode reports the workbook's date system (0 = 1900, 1 = 1904).
func (w *WorkBook) DateMode() uint16 { return w.dateMode }

// FormatText returns the format string behind a format index, or "".
func (w *WorkBook) FormatText(no uint16) string {
	if f := w.Formats[no]; f != nil {
		return f.str
	}
	return ""
}

// FormatIndexForXf returns the format index behind an XF index.
func (w *WorkBook) FormatIndexForXf(xf int) uint16 {
	if xf >= 0 && xf < len(w.Xfs) {
		return w.Xfs[xf].formatNo()
	}
	return 0
}

// ErrorText renders an Excel error code the way xlrd's error_text_from_code
// does, so both sides produce the same string for an error cell.
func ErrorText(code int) string {
	if s, ok := errorTextFromCode[code]; ok {
		return s
	}
	return "#ERR" + strconv.Itoa(code)
}

var errorTextFromCode = map[int]string{
	0: "#NULL!", 7: "#DIV/0!", 15: "#VALUE!", 23: "#REF!",
	29: "#NAME?", 36: "#NUM!", 42: "#N/A",
}

// Cell is one cell's value with the format index the date rule needs.
type Cell struct {
	Present  bool
	Kind     string // number | text | bool | error | blank
	Number   float64
	Text     string
	Bool     bool
	Error    int
	FormatNo uint16
}

// cellAt resolves the handler owning (row, col) and the offset within it, since
// MULRK and MULBLANK carry several cells in one handler.
func (r *Row) cellAt(col int) (contentHandler, int) {
	serial := uint16(col)
	if ch, ok := r.cols[serial]; ok {
		return ch, 0
	}
	for _, v := range r.cols {
		if v.FirstCol() <= serial && v.LastCol() >= serial {
			return v, col - int(v.FirstCol())
		}
	}
	return nil, 0
}

// TypedCell returns the cell at (row, col) with its value and format index.
func (w *WorkBook) TypedCell(sheet, row, col int) Cell {
	sh := w.GetSheet(sheet)
	if sh == nil {
		return Cell{}
	}
	r := sh.Row(row)
	if r == nil {
		return Cell{}
	}
	ch, off := r.cellAt(col)
	if ch == nil {
		return Cell{Present: true, Kind: "blank"}
	}
	return w.typedFrom(ch, off)
}

func (w *WorkBook) typedFrom(ch contentHandler, off int) Cell {
	switch c := ch.(type) {
	case *BoolErrCol:
		if c.IsError != 0 {
			return Cell{Present: true, Kind: "error", Error: int(c.Value)}
		}
		return Cell{Present: true, Kind: "bool", Bool: c.Value != 0}
	case *NumberCol:
		return Cell{Present: true, Kind: "number", Number: c.Float, FormatNo: w.FormatIndexForXf(int(c.Index))}
	case *RkCol:
		return w.rkCell(c.Xfrk)
	case *MulrkCol:
		if off < len(c.Xfrks) {
			return w.rkCell(c.Xfrks[off])
		}
		return Cell{Present: true, Kind: "blank"}
	case *MulBlankCol, *BlankCol:
		return Cell{Present: true, Kind: "blank"}
	case *FormulaCol:
		value, kind := c.Cached()
		out := Cell{Present: true, Kind: kind, FormatNo: w.FormatIndexForXf(int(c.Header.IndexXf))}
		switch kind {
		case "number":
			out.Number = value.(float64)
		case "bool":
			out.Bool = value.(bool)
		case "error":
			out.Error = value.(int)
		case "text":
			out.Text, _ = value.(string)
		}
		return out
	case *FormulaStringCol:
		return Cell{Present: true, Kind: "text", Text: c.RenderedValue}
	case *LabelsstCol:
		s := ""
		if int(c.Sst) < len(w.sst) {
			s = w.sst[int(c.Sst)]
		}
		return Cell{Present: true, Kind: "text", Text: s, FormatNo: w.FormatIndexForXf(int(c.Xf))}
	case *labelCol:
		return Cell{Present: true, Kind: "text", Text: c.Str, FormatNo: w.FormatIndexForXf(int(c.BlankCol.Xf))}
	}
	return Cell{Present: true, Kind: "blank"}
}

func (w *WorkBook) rkCell(xf XfRk) Cell {
	i, f, isFloat := xf.Rk.number()
	if !isFloat {
		f = float64(i)
	}
	return Cell{Present: true, Kind: "number", Number: f, FormatNo: w.FormatIndexForXf(int(xf.Index))}
}

// IsDate decides whether a cell with this format index is a date, by xlrd's
// rule: a format record's string decides when the workbook carries one, and the
// built-in table is the fallback for a key with no record.
func (w *WorkBook) IsDate(formatNo uint16) bool {
	if txt := w.FormatText(formatNo); txt != "" {
		return IsDateFormatString(txt)
	}
	return stdFormatType(int(formatNo)) == fmtDate
}

// ── xlrd's rule, ported from xlrd/formatting.py ─────────────────────────────

const (
	fmtUnknown = iota
	fmtDate
	fmtNumber
	fmtGeneral
	fmtText
)

var stdFormatRanges = []struct{ lo, hi, ty int }{
	{0, 0, fmtGeneral}, {1, 13, fmtNumber}, {14, 22, fmtDate},
	{27, 36, fmtDate}, {37, 44, fmtNumber}, {45, 47, fmtDate},
	{48, 48, fmtNumber}, {49, 49, fmtText}, {50, 58, fmtDate},
	{59, 62, fmtNumber}, {67, 70, fmtNumber}, {71, 81, fmtDate},
}

func stdFormatType(key int) int {
	for _, r := range stdFormatRanges {
		if key >= r.lo && key <= r.hi {
			return r.ty
		}
	}
	return fmtUnknown
}

var bracketed = regexp.MustCompile(`\[[^]]*\]`)

var nonDateFormats = map[string]bool{
	"0.00E+00": true, "##0.0E+0": true,
	"General": true, "GENERAL": true, "general": true, "@": true,
}

// IsDateFormatString applies xlrd's format-string rule: whether a number-format
// string denotes a date. It is exported because the rule is usable on its own —
// IsDate reaches it only for a workbook whose format record it can read.
func IsDateFormatString(format string) bool {
	var b strings.Builder
	state := 0
	for _, c := range format {
		switch state {
		case 0:
			switch {
			case c == '"':
				state = 1
			case c == '\\' || c == '_' || c == '*':
				state = 2
			case strings.ContainsRune("$-+/(): ", c):
			default:
				b.WriteRune(c)
			}
		case 1:
			if c == '"' {
				state = 0
			}
		case 2:
			state = 0
		}
	}
	s := bracketed.ReplaceAllString(b.String(), "")
	if nonDateFormats[s] {
		return false
	}
	dateCount, numCount := 0, 0
	for _, c := range s {
		switch {
		case strings.ContainsRune("ymdhsYMDHS", c):
			dateCount += 5
		case strings.ContainsRune("0#?", c):
			numCount += 5
		}
	}
	if dateCount > 0 && numCount == 0 {
		return true
	}
	if numCount > 0 && dateCount == 0 {
		return false
	}
	return dateCount > numCount
}
