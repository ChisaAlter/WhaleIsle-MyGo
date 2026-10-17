package workspacefs

import (
	"strconv"
	"strings"
)

// This is saved presentation metadata, not an executable formula or a workbook
// rendering engine. Missing metadata stays absent, rather than being invented.
type XLSXCellStyle struct {
	Available            bool
	FontName             string
	FontSize             float32
	Bold, Italic         bool
	Horizontal, Vertical string
	Wrap                 bool
}
type xlsxStyleFlag struct {
	Value string `xml:"val,attr"`
}

func (f *xlsxStyleFlag) enabled() bool {
	return f != nil && f.Value != "0" && !strings.EqualFold(f.Value, "false")
}

type xlsxSourceFont struct {
	Name struct {
		Value string `xml:"val,attr"`
	} `xml:"name"`
	Size struct {
		Value float32 `xml:"val,attr"`
	} `xml:"sz"`
	Bold   *xlsxStyleFlag `xml:"b"`
	Italic *xlsxStyleFlag `xml:"i"`
}
type xlsxSourceAlignment struct {
	Horizontal string `xml:"horizontal,attr"`
	Vertical   string `xml:"vertical,attr"`
	Wrap       string `xml:"wrapText,attr"`
}
type xlsxSourceXF struct {
	Base           *int                 `xml:"xfId,attr"`
	ApplyFont      string               `xml:"applyFont,attr"`
	ApplyFormat    string               `xml:"applyNumberFormat,attr"`
	ApplyAlignment string               `xml:"applyAlignment,attr"`
	Format         *int                 `xml:"numFmtId,attr"`
	Font           *int                 `xml:"fontId,attr"`
	Alignment      *xlsxSourceAlignment `xml:"alignment"`
}

func savedXLSXStyle(fonts []xlsxSourceFont, xf xlsxSourceXF) XLSXCellStyle {
	s := XLSXCellStyle{Available: true}
	if xf.Alignment != nil {
		s.Horizontal = xf.Alignment.Horizontal
		s.Vertical = xf.Alignment.Vertical
		s.Wrap = xf.Alignment.Wrap == "1" || strings.EqualFold(xf.Alignment.Wrap, "true")
	}
	fontIndex := 0
	if xf.Font != nil {
		fontIndex = *xf.Font
	}
	if fontIndex >= 0 && fontIndex < len(fonts) {
		font := fonts[fontIndex]
		s.FontName = font.Name.Value
		s.FontSize = font.Size.Value
		s.Bold = font.Bold.enabled()
		s.Italic = font.Italic.enabled()
	}
	return s
}
func (f workbookFormats) presentation(index int) XLSXCellStyle {
	if index >= 0 && index < len(f.presentationStyles) {
		return f.presentationStyles[index]
	}
	return XLSXCellStyle{}
}

// Apply the saved cell style base only where the cell XF omits/declines a group.
func inheritedXLSXXF(xf xlsxSourceXF, bases []xlsxSourceXF) xlsxSourceXF {
	if xf.Base == nil || *xf.Base < 0 || *xf.Base >= len(bases) {
		return xf
	}
	base := bases[*xf.Base]
	if xf.Font == nil || xf.ApplyFont == "0" || xf.ApplyFont == "false" {
		xf.Font = base.Font
	}
	if xf.Format == nil || xf.ApplyFormat == "0" || xf.ApplyFormat == "false" {
		xf.Format = base.Format
	}
	if xf.Alignment == nil || xf.ApplyAlignment == "0" || xf.ApplyAlignment == "false" {
		xf.Alignment = base.Alignment
	}
	return xf
}
func savedXLSXMerge(ref string) (XLSXMerge, bool) {
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return XLSXMerge{}, false
	}
	coordinate := func(s string) (int, int, bool) {
		col := colIndex(s)
		n, err := strconv.Atoi(strings.TrimLeft(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
		return n - 1, col, err == nil && col >= 0 && col < 16384 && n > 0 && n <= 1048576
	}
	r, c, ok := coordinate(parts[0])
	r2, c2, ok2 := coordinate(parts[1])
	return XLSXMerge{r, c, r2, c2}, ok && ok2 && r2 >= r && c2 >= c
}
