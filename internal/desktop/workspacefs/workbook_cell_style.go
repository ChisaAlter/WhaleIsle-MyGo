package workspacefs

import "strings"

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
	Format    int                 `xml:"numFmtId,attr"`
	Font      int                 `xml:"fontId,attr"`
	Alignment xlsxSourceAlignment `xml:"alignment"`
}

func savedXLSXStyle(fonts []xlsxSourceFont, xf xlsxSourceXF) XLSXCellStyle {
	s := XLSXCellStyle{Available: true, Horizontal: xf.Alignment.Horizontal, Vertical: xf.Alignment.Vertical, Wrap: xf.Alignment.Wrap == "1" || strings.EqualFold(xf.Alignment.Wrap, "true")}
	if xf.Font >= 0 && xf.Font < len(fonts) {
		font := fonts[xf.Font]
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
