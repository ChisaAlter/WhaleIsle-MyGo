package files

import (
	"encoding/hex"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/workspacefs"
)

func tablePaintColor(value string) (ui.Color, bool) {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		return ui.Color{}, false
	}
	data, err := hex.DecodeString(value)
	if err != nil {
		return ui.Color{}, false
	}
	return ui.RGB(data[0], data[1], data[2]), true
}
func tableBorderPattern(style string) (float32, []float32, bool) {
	switch style {
	case "hair":
		return .5, nil, true
	case "thin":
		return 1, nil, true
	case "medium":
		return 2, nil, true
	case "thick":
		return 3, nil, true
	case "double":
		return 1, nil, true
	case "dashed":
		return 1, []float32{3, 2}, true
	case "mediumDashed":
		return 2, []float32{3, 2}, true
	case "dotted":
		return 1, []float32{1, 2}, true
	case "dashDot":
		return 1, []float32{3, 2, 1, 2}, true
	case "mediumDashDot", "slantDashDot":
		return 2, []float32{3, 2, 1, 2}, style != "slantDashDot"
	case "dashDotDot":
		return 1, []float32{3, 2, 1, 2, 1, 2}, true
	case "mediumDashDotDot":
		return 2, []float32{3, 2, 1, 2, 1, 2}, true
	}
	return 0, nil, false
}

// tableFillPattern maps the OOXML pattern fills that have a direct native
// Stripes equivalent. Cross-hatch and gray patterns intentionally return false:
// pretending a single stripe direction is equivalent would hide source paint
// differences. The widths are deterministic fallbacks for the OOXML dark/light
// density families; the saved foreground/background colours are never guessed.
func tableFillPattern(style workspacefs.XLSXCellStyle) (ui.Color, ui.Color, float32, float32, float32, bool) {
	if style.FillPattern == "" || style.FillPattern == "none" || style.FillPattern == "solid" {
		return ui.Color{}, ui.Color{}, 0, 0, 0, false
	}
	background, backgroundOK := tablePaintColor(style.FillColor)
	foreground, foregroundOK := tablePaintColor(style.FillColor2)
	if !backgroundOK || !foregroundOK {
		return ui.Color{}, ui.Color{}, 0, 0, 0, false
	}
	width, gap, angle := float32(1), float32(3), float32(0)
	switch style.FillPattern {
	case "lightVertical":
		angle = 0
	case "lightHorizontal":
		angle = 90
	case "lightDown":
		angle = 45
	case "lightUp":
		angle = 135
	case "darkVertical":
		width, gap, angle = 3, 2, 0
	case "darkHorizontal":
		width, gap, angle = 3, 2, 90
	case "darkDown":
		width, gap, angle = 3, 2, 45
	case "darkUp":
		width, gap, angle = 3, 2, 135
	default:
		return ui.Color{}, ui.Color{}, 0, 0, 0, false
	}
	return background, foreground, width, gap, angle, true
}
func tablePaintUnsupported(style workspacefs.XLSXCellStyle) bool {
	if style.PaintUnsupported {
		return true
	}
	if style.FillPattern != "" && style.FillPattern != "none" && style.FillPattern != "solid" {
		if _, _, _, _, _, ok := tableFillPattern(style); !ok {
			return true
		}
	}
	for _, edge := range style.Borders {
		if edge.Style != "" && edge.Style != "none" {
			if _, _, ok := tableBorderPattern(edge.Style); !ok {
				return true
			}
		}
	}
	return false
}
func paintTableSourceStyle(button ui.Element, style workspacefs.XLSXCellStyle, selected bool) {
	// Selection retains the native highlight instead of hiding it under source fill.
	if !selected {
		if style.FillPattern == "solid" {
			if color, ok := tablePaintColor(style.FillColor); ok {
				button.Background(color)
			}
		} else if background, foreground, width, gap, angle, ok := tableFillPattern(style); ok {
			button.Background(background).Stripes(foreground, width, gap, angle)
		}
	}
	button.DrawOver(func(p *ui.Painter, r ui.Rect) {
		for side, edge := range style.Borders {
			width, pattern, supported := tableBorderPattern(edge.Style)
			color, ok := tablePaintColor(edge.Color)
			if !ok || !supported {
				continue
			}
			horizontal := side == 0 || side == 2
			x, y, length := r.X, r.Y, r.W
			if !horizontal {
				length = r.H
			}
			if side == 1 {
				x = r.X + r.W - width/2
			}
			if side == 2 {
				y = r.Y + r.H - width/2
			}
			if side == 0 {
				y += width / 2
			}
			if side == 3 {
				x += width / 2
			}
			line := func(start, end, offset float32) {
				if horizontal {
					p.Line(x+start, y+offset, x+end, y+offset, width, color)
				} else {
					p.Line(x+offset, y+start, x+offset, y+end, width, color)
				}
			}
			if len(pattern) == 0 {
				line(0, length, 0)
			} else {
				position, index := float32(0), 0
				for position < length {
					next := min(length, position+pattern[index%len(pattern)]*width)
					if index%2 == 0 {
						line(position, next, 0)
					}
					position = next
					index++
				}
			}
			if edge.Style == "double" {
				offset := float32(2)
				if side == 1 || side == 2 {
					offset = -2
				}
				line(0, length, offset)
			}
		}
	})
}
