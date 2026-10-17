package workspacefs

import (
	"archive/zip"
	"context"
	"encoding/hex"
	"encoding/xml"
	"io"
	"strings"
)

// Edges are top/right/bottom/left. Colors are canonical saved RGB, not UI theme
// approximations. Unresolved source features are disclosed rather than guessed.
type XLSXBorder struct{ Style, Color string }
type xlsxSourceColor struct {
	RGB     string  `xml:"rgb,attr"`
	Theme   *int    `xml:"theme,attr"`
	Indexed *int    `xml:"indexed,attr"`
	Tint    float64 `xml:"tint,attr"`
	Auto    bool    `xml:"auto,attr"`
}
type xlsxSourceFill struct {
	Pattern struct {
		Type       string          `xml:"patternType,attr"`
		Foreground xlsxSourceColor `xml:"fgColor"`
	} `xml:"patternFill"`
	Gradient *struct{} `xml:"gradientFill"`
}
type xlsxSourceEdge struct {
	Style string          `xml:"style,attr"`
	Color xlsxSourceColor `xml:"color"`
}
type xlsxSourceBorder struct {
	Top      xlsxSourceEdge `xml:"top"`
	Right    xlsxSourceEdge `xml:"right"`
	Bottom   xlsxSourceEdge `xml:"bottom"`
	Left     xlsxSourceEdge `xml:"left"`
	Diagonal xlsxSourceEdge `xml:"diagonal"`
}

func savedXLSXColor(source xlsxSourceColor, theme map[int]string, palette []xlsxSourceColor) (string, bool) {
	if source.Auto || source.Tint != 0 {
		return "", false
	}
	raw := source.RGB
	if raw == "" && source.Theme != nil {
		raw = theme[*source.Theme]
	}
	if raw == "" && source.Indexed != nil && *source.Indexed >= 0 && *source.Indexed < len(palette) {
		raw = palette[*source.Indexed].RGB
	}
	if len(raw) != 6 && len(raw) != 8 {
		return "", false
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return "", false
	}
	if len(raw) == 8 {
		raw = raw[2:]
	} // Spreadsheet colors use opaque RGB; the saved alpha byte is ignored.
	if len(raw) != 6 {
		return "", false
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return "", false
	}
	return "#" + strings.ToUpper(raw), true
}
func readXLSXTheme(ctx context.Context, zr *zip.Reader, seen map[string]bool) (map[int]string, error) {
	colors := map[int]string{}
	if !seen["xl/theme/theme1.xml"] {
		return colors, nil
	}
	rc, dec, err := xlsxPart(ctx, zr, "xl/theme/theme1.xml")
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	// A clrScheme's child names define their meaning; order alone is insufficient.
	for {
		token, e := dec.Token()
		if e != nil {
			if e == io.EOF {
				return colors, nil
			}
			return nil, e
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "clrScheme" {
			continue
		}
		var scheme struct {
			Entries []struct {
				XMLName xml.Name
				RGB     struct {
					Value string `xml:"val,attr"`
				} `xml:"srgbClr"`
				System struct {
					Last string `xml:"lastClr,attr"`
				} `xml:"sysClr"`
			} `xml:",any"`
		}
		if err := dec.DecodeElement(&scheme, &start); err != nil {
			return nil, err
		}
		slots := map[string]int{"lt1": 0, "dk1": 1, "lt2": 2, "dk2": 3, "accent1": 4, "accent2": 5, "accent3": 6, "accent4": 7, "accent5": 8, "accent6": 9, "hlink": 10, "folHlink": 11}
		for _, entry := range scheme.Entries {
			if index, ok := slots[entry.XMLName.Local]; ok {
				value := entry.RGB.Value
				if value == "" {
					value = entry.System.Last
				}
				colors[index] = value
			}
		}
		return colors, nil
	}
}
func savedXLSXPaint(style *XLSXCellStyle, xf xlsxSourceXF, fills []xlsxSourceFill, borders []xlsxSourceBorder, theme map[int]string, palette []xlsxSourceColor) {
	if xf.Fill != nil && *xf.Fill >= 0 && *xf.Fill < len(fills) {
		fill := fills[*xf.Fill]
		style.FillPattern = fill.Pattern.Type
		if fill.Pattern.Type == "solid" {
			var ok bool
			style.FillColor, ok = savedXLSXColor(fill.Pattern.Foreground, theme, palette)
			style.PaintUnsupported = !ok
		} else if fill.Gradient != nil || (fill.Pattern.Type != "" && fill.Pattern.Type != "none") {
			style.PaintUnsupported = true
		}
	}
	if xf.Border != nil && *xf.Border >= 0 && *xf.Border < len(borders) {
		border := borders[*xf.Border]
		for i, edge := range []xlsxSourceEdge{border.Top, border.Right, border.Bottom, border.Left} {
			if edge.Style == "" || edge.Style == "none" {
				continue
			}
			color, ok := savedXLSXColor(edge.Color, theme, palette)
			style.Borders[i] = XLSXBorder{edge.Style, color}
			style.PaintUnsupported = style.PaintUnsupported || !ok
		}
		if border.Diagonal.Style != "" && border.Diagonal.Style != "none" {
			style.PaintUnsupported = true
		}
	}
}
