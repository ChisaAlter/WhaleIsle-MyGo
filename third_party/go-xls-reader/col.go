package xls

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"time"

	yymmdd "github.com/extrame/goyymmdd"
)

// content type
type contentHandler interface {
	String(*WorkBook) []string
	FirstCol() uint16
	LastCol() uint16
}

type Col struct {
	RowB      uint16
	FirstColB uint16
}

type Coler interface {
	Row() uint16
}

func (c *Col) Row() uint16 {
	return c.RowB
}

func (c *Col) FirstCol() uint16 {
	return c.FirstColB
}

func (c *Col) LastCol() uint16 {
	return c.FirstColB
}

func (c *Col) String(wb *WorkBook) []string {
	return []string{"default"}
}

type XfRk struct {
	Index uint16
	Rk    RK
}

func (xf *XfRk) String(wb *WorkBook) string {
	idx := int(xf.Index)
	if len(wb.Xfs) > idx {
		fNo := wb.Xfs[idx].formatNo()
		if fNo >= 164 { // user defined format
			if formatter := wb.Formats[fNo]; formatter != nil {
				formatterLower := strings.ToLower(formatter.str)
				if formatterLower == "general" ||
					strings.Contains(formatter.str, "#") ||
					strings.Contains(formatter.str, ".00") ||
					strings.Contains(formatterLower, "m/y") ||
					strings.Contains(formatterLower, "d/y") ||
					strings.Contains(formatterLower, "m.y") ||
					strings.Contains(formatterLower, "d.y") ||
					strings.Contains(formatterLower, "h:") ||
					strings.Contains(formatterLower, "д.г") {
					//If format contains # or .00 then this is a number
					return xf.Rk.String()
				} else {
					i, f, isFloat := xf.Rk.number()
					if !isFloat {
						f = float64(i)
					}
					t := TimeFromExcelTime(f, wb.dateMode == 1)
					return yymmdd.Format(t, formatter.str)
				}
			}
			// see http://www.openoffice.org/sc/excelfileformat.pdf Page #174
		} else if 14 <= fNo && fNo <= 17 || fNo == 22 || 27 <= fNo && fNo <= 36 || 50 <= fNo && fNo <= 58 { // jp. date format
			i, f, isFloat := xf.Rk.number()
			if !isFloat {
				f = float64(i)
			}
			t := TimeFromExcelTime(f, wb.dateMode == 1)
			return t.Format(time.RFC3339) //TODO it should be international
		}
	}
	return xf.Rk.String()
}

type RK uint32

func (rk RK) number() (intNum int64, floatNum float64, isFloat bool) {
	multiplied := rk & 1
	isInt := rk & 2
	val := int32(rk) >> 2
	if isInt == 0 {
		isFloat = true
		floatNum = math.Float64frombits(uint64(val) << 34)
		if multiplied != 0 {
			floatNum = floatNum / 100
		}
		return
	}
	if multiplied != 0 {
		isFloat = true
		floatNum = float64(val) / 100
		return
	}
	return int64(val), 0, false
}

func (rk RK) String() string {
	i, f, isFloat := rk.number()
	if isFloat {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatInt(i, 10)
}

var ErrIsInt = fmt.Errorf("is int")

func (rk RK) Float() (float64, error) {
	_, f, isFloat := rk.number()
	if !isFloat {
		return 0, ErrIsInt
	}
	return f, nil
}

type MulrkCol struct {
	Col
	Xfrks    []XfRk
	LastColB uint16
}

func (c *MulrkCol) LastCol() uint16 {
	return c.LastColB
}

func (c *MulrkCol) String(wb *WorkBook) []string {
	var res = make([]string, len(c.Xfrks))
	for i := 0; i < len(c.Xfrks); i++ {
		xfrk := c.Xfrks[i]
		res[i] = xfrk.String(wb)
	}
	return res
}

type MulBlankCol struct {
	Col
	Xfs      []uint16
	LastColB uint16
}

func (c *MulBlankCol) LastCol() uint16 {
	return c.LastColB
}

func (c *MulBlankCol) String(wb *WorkBook) []string {
	return make([]string, len(c.Xfs))
}

type NumberCol struct {
	Col
	Index uint16
	Float float64
}

func (c *NumberCol) String(wb *WorkBook) []string {
	if fNo := wb.Xfs[c.Index].formatNo(); fNo != 0 {
		t := TimeFromExcelTime(c.Float, wb.dateMode == 1)
		return []string{yymmdd.Format(t, wb.Formats[fNo].str)}
	}
	return []string{strconv.FormatFloat(c.Float, 'f', -1, 64)}
}

type FormulaStringCol struct {
	Col
	RenderedValue string
}

func (c *FormulaStringCol) String(wb *WorkBook) []string {
	return []string{c.RenderedValue}
}

//str, err = wb.get_string(buf_item, size)
//wb.sst[offset_pre] = wb.sst[offset_pre] + str

type FormulaCol struct {
	Header struct {
		Col
		IndexXf uint16
		Result  [8]byte
		Flags   uint16
		_       uint32
	}
	Bts []byte
}

func (c *FormulaCol) Row() uint16      { return c.Header.RowB }
func (c *FormulaCol) FirstCol() uint16 { return c.Header.FirstColB }
func (c *FormulaCol) LastCol() uint16  { return c.Header.FirstColB }

// Cached decodes the result a FORMULA record carries: the value Excel last
// computed for the formula, which is what a reader has to report for a formula
// cell (the formula itself is not evaluated). Mirrors xlrd's handling — the
// last two bytes being 0xFFFF mark a non-numeric result, selected by the first
// byte; otherwise the eight bytes are an IEEE double.
//
// kind is "number", "bool", "error", "text" or "blank". A string result is
// not available here: it arrives in the STRING record that follows, which the
// parser turns into a FormulaStringCol at the same cell.
func (c *FormulaCol) Cached() (value interface{}, kind string) {
	res := c.Header.Result
	if res[6] == 0xFF && res[7] == 0xFF {
		switch res[0] {
		case 0: // a STRING record follows, handled as FormulaStringCol
			return "", "text"
		case 1:
			return res[2] != 0, "bool"
		case 2:
			return int(res[2]), "error"
		case 3: // an empty (zero-length) string, not an empty cell
			return "", "text"
		}
		return nil, "blank"
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(res[:])), "number"
}

func (c *FormulaCol) String(wb *WorkBook) []string {
	value, kind := c.Cached()
	switch kind {
	case "number":
		return []string{strconv.FormatFloat(value.(float64), 'f', -1, 64)}
	case "bool":
		if value.(bool) {
			return []string{"TRUE"}
		}
		return []string{"FALSE"}
	case "error":
		return []string{ErrorText(value.(int))}
	}
	return []string{""}
}

type RkCol struct {
	Col
	Xfrk XfRk
}

func (c *RkCol) String(wb *WorkBook) []string {
	return []string{c.Xfrk.String(wb)}
}

type LabelsstCol struct {
	Col
	Xf  uint16
	Sst uint32
}

func (c *LabelsstCol) String(wb *WorkBook) []string {
	return []string{wb.sst[int(c.Sst)]}
}

type labelCol struct {
	BlankCol
	Str string
}

func (c *labelCol) String(wb *WorkBook) []string {
	return []string{c.Str}
}

type BlankCol struct {
	Col
	Xf uint16
}

func (c *BlankCol) String(wb *WorkBook) []string {
	return []string{""}
}

// BoolErrCol holds the stored boolean/error value in a BIFF BOOLERR record.
type BoolErrCol struct {
	BlankCol
	Value   byte
	IsError byte
}

func (c *BoolErrCol) String(wb *WorkBook) []string {
	if c.IsError != 0 {
		return []string{ErrorText(int(c.Value))}
	}
	if c.Value != 0 {
		return []string{"TRUE"}
	}
	return []string{"FALSE"}
}
