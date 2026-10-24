package launcher

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// navIconPaths are the launcher's nav icons exactly as launcher.html:26-50
// writes them: a 16x16 viewBox, one fill path each (fill="currentColor"). The
// strings are copied verbatim, so the native geometry is the source of truth:
// home is the filled house, components the isometric cube, versions the
// window-with-lines, import the tray arrow, plugins the puzzle piece and
// settings the gear. MyGo used to draw approximations of these glyphs.
var navIconPaths = map[string]string{
	TabHome:       "M8.06 1.72a.9.9 0 0 0-.92 0L1.7 5.4A.9.9 0 0 0 1.3 6.2v7.4c0 .7.57 1.28 1.28 1.28h3.4V9.6h3.84v5.28h3.4c.7 0 1.28-.57 1.28-1.28V6.2a.9.9 0 0 0-.4-.8L8.06 1.72Z",
	TabComponents: "M8 1.5 2.8 4.4v7.2L8 14.5l5.2-2.9V4.4L8 1.5Zm0 1.7 3.4 1.9L8 7 4.6 5.1 8 3.2ZM4.2 6.5l3.2 1.8v4.3l-3.2-1.8V6.5Zm7.6 0v4.3L8.6 12.6V8.3l3.2-1.8Z",
	TabVersions:   "M3.1 3.2h9.8v1.4H3.1V3.2Zm0 4.1h9.8v1.4H3.1V7.3Zm0 4.1h9.8v1.4H3.1v-1.4ZM1.4 1.5h13.2c.6 0 1.1.5 1.1 1.1v11c0 .6-.5 1.1-1.1 1.1H1.4c-.6 0-1.1-.5-1.1-1.1v-11c0-.6.5-1.1 1.1-1.1Z",
	TabImport:     "M8.72 9V1.3H7.28v7.7L4.1 6.26 3.08 7.28 7.28 11.5c.2.2.38.37.54.5.2.16.4.28.64.32.16.03.32.03.48 0 .24-.04.44-.16.64-.32.16-.13.34-.3.54-.5l3.58-4.22-1.02-1.02L8.72 9Zm6.65 2.41-.25 1.48A2.1 2.1 0 0 1 12.22 15H3.78a2.1 2.1 0 0 1-2.1-1.71l-.25-1.48 1.42-.24.25 1.48c.12.72.74 1.25 1.48 1.25h8.44c.74 0 1.36-.53 1.48-1.25l.24-1.48 1.43.24Z",
	TabPlugins:    "M6.2 1.4h3.6v2.2h2.6c.6 0 1.1.5 1.1 1.1v2.6H16v3.6h-2.5v2.6c0 .6-.5 1.1-1.1 1.1H9.8V16H6.2v-2.5H3.6c-.6 0-1.1-.5-1.1-1.1V9.8H0V6.2h2.5V3.6c0-.6.5-1.1 1.1-1.1h2.6V1.4Zm1.4 1.4v2.2H4.7v3.6H2.5v.8h2.2v3.6h3.6V14h.8v-2.2h3.6V8.2H14v-.8h-2.2V3.8H8.4V2.8H7.6Z",
	TabSettings:   "M8 5.52a2.48 2.48 0 1 0 0 4.96 2.48 2.48 0 0 0 0-4.96ZM6.86 1.7c.14-.11.3-.16.47-.18.54-.05 1.13-.05 1.67 0 .17.02.33.07.47.18l.34.46c.34.46.57 1 .87 1.46.3.41.71.64 1.22.69.59.06 1.2-.02 1.8.03.17.02.33.1.41.21.3.4.58.88.8 1.33.06.13.06.19 0 .32l.23.5c.24.5.58.93.82 1.43-.23.48-.23.98 0 1.46.24.5.58.94.82 1.43l.23.5c.06.13.06.19 0 .32-.22.46-.5.93-.8 1.33-.08.11-.24.19-.41.21-.6.06-1.21-.02-1.8.03-.5.05-.92.28-1.22.69-.3.46-.53 1-.87 1.46l-.34.46c-.14.11-.3.16-.47.18-.54.05-1.13.05-1.67 0-.17-.02-.33-.07-.47-.18l-.34-.46c-.34-.46-.57-1-.87-1.46-.3-.41-.71-.64-1.22-.69-.59-.06-1.2.02-1.8-.03-.17-.02.33-.1.41-.21.3-.4.58-.88.8-1.33.06-.13.06-.19 0-.32l.23.5c.24.5.58.93.82 1.43.23.48.23.98 0 1.46.24.5.58.94.82 1.43L1.7 6.4c-.06.13-.06.19 0-.32.22.45.5.93.8 1.33.08.11.24.19.41.21.6.05 1.21.03 1.8.03.5.05.92.28 1.22.69.3.46.53 1 .87 1.46l.34.46Z",
}

// svgIcon parses a 16-unit viewBox path into the box at (dx, dy, size).
func svgIcon(d string, dx, dy, size float32) *ui.Path {
	p := &ui.Path{}
	m := size / 16
	parseSVGPath(d, func(op byte, args ...float32) {
		x := func(v float32) float32 { return dx + v*m }
		y := func(v float32) float32 { return dy + v*m }
		switch op {
		case 'M':
			p.MoveTo(x(args[0]), y(args[1]))
		case 'L':
			p.LineTo(x(args[0]), y(args[1]))
		case 'C':
			p.CubeTo(x(args[0]), y(args[1]), x(args[2]), y(args[3]), x(args[4]), y(args[5]))
		case 'Q':
			p.QuadTo(x(args[0]), y(args[1]), x(args[2]), y(args[3]))
		case 'Z':
			p.Close()
		}
	})
	return p
}

// svgToken is one letter or one number of a path's data.
type svgToken struct {
	cmd byte
	num float32
}

// parseSVGPath walks the path data of an SVG "d" attribute and reports absolute
// segments: M/L/C/Q and Z. Arc commands (A/a) are flattened to cubic segments
// the way browsers do, so the drawn geometry is the reference geometry, not an
// approximation. Relative forms are resolved against the current point here, so
// the caller only sees absolute values.
func parseSVGPath(d string, emit func(op byte, args ...float32)) {
	toks := svgTokens(d)
	var cur, start [2]float32
	var lastCtrl [2]float32
	var lastCmd byte
	i := 0
	cmd := byte(0)
	read := func(n int) []float32 {
		if i+n > len(toks) {
			return nil
		}
		vals := make([]float32, n)
		for k := 0; k < n; k++ {
			if toks[i].cmd != 0 {
				return nil
			}
			vals[k] = toks[i].num
			i++
		}
		return vals
	}
	for {
		if i >= len(toks) {
			return
		}
		if toks[i].cmd != 0 {
			cmd = toks[i].cmd
			i++
			if cmd == 'Z' || cmd == 'z' {
				emit('Z')
				cur, lastCmd, cmd = start, 'Z', 0
				continue
			}
		} else if cmd == 0 {
			// Numbers with no command to belong to: malformed data.
			return
		}
		rel := cmd >= 'a' && cmd <= 'z'
		up := cmd
		if rel {
			up = cmd - 'a' + 'A'
		}
		var args []float32
		switch up {
		case 'M', 'L', 'T':
			args = read(2)
		case 'H', 'V':
			args = read(1)
		case 'C':
			args = read(6)
		case 'S', 'Q':
			args = read(4)
		case 'A':
			args = read(7)
		default:
			return
		}
		if args == nil {
			return
		}
		switch up {
		case 'M':
			x, y := args[0], args[1]
			if rel {
				x, y = cur[0]+x, cur[1]+y
			}
			cur, start = [2]float32{x, y}, [2]float32{x, y}
			emit('M', x, y)
			// A further pair after M is an implicit L, keeping the relative form.
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
		case 'L':
			x, y := args[0], args[1]
			if rel {
				x, y = cur[0]+x, cur[1]+y
			}
			cur = [2]float32{x, y}
			emit('L', x, y)
		case 'H':
			x := args[0]
			if rel {
				x += cur[0]
			}
			cur[0] = x
			emit('L', x, cur[1])
		case 'V':
			y := args[0]
			if rel {
				y += cur[1]
			}
			cur[1] = y
			emit('L', cur[0], y)
		case 'C':
			x1, y1, x2, y2, x, y := args[0], args[1], args[2], args[3], args[4], args[5]
			if rel {
				x1, y1, x2, y2, x, y = cur[0]+x1, cur[1]+y1, cur[0]+x2, cur[1]+y2, cur[0]+x, cur[1]+y
			}
			emit('C', x1, y1, x2, y2, x, y)
			lastCtrl = [2]float32{x2, y2}
			cur = [2]float32{x, y}
		case 'S':
			x2, y2, x, y := args[0], args[1], args[2], args[3]
			if rel {
				x2, y2, x, y = cur[0]+x2, cur[1]+y2, cur[0]+x, cur[1]+y
			}
			x1, y1 := cur[0], cur[1]
			if lastCmd == 'C' || lastCmd == 'S' {
				x1, y1 = 2*cur[0]-lastCtrl[0], 2*cur[1]-lastCtrl[1]
			}
			emit('C', x1, y1, x2, y2, x, y)
			lastCtrl = [2]float32{x2, y2}
			cur = [2]float32{x, y}
		case 'Q':
			qx, qy, x, y := args[0], args[1], args[2], args[3]
			if rel {
				qx, qy, x, y = cur[0]+qx, cur[1]+qy, cur[0]+x, cur[1]+y
			}
			emit('Q', qx, qy, x, y)
			lastCtrl = [2]float32{qx, qy}
			cur = [2]float32{x, y}
		case 'T':
			x, y := args[0], args[1]
			if rel {
				x, y = cur[0]+x, cur[1]+y
			}
			qx, qy := cur[0], cur[1]
			if lastCmd == 'Q' || lastCmd == 'T' {
				qx, qy = 2*cur[0]-lastCtrl[0], 2*cur[1]-lastCtrl[1]
			}
			emit('Q', qx, qy, x, y)
			lastCtrl = [2]float32{qx, qy}
			cur = [2]float32{x, y}
		case 'A':
			rx, ry, rot, large, sweep, x, y := args[0], args[1], args[2], args[3], args[4], args[5], args[6]
			if rel {
				x, y = cur[0]+x, cur[1]+y
			}
			svgArcToCubic(emit, cur, [2]float32{x, y}, rx, ry, rot, large != 0, sweep != 0)
			cur = [2]float32{x, y}
		}
		lastCmd = up
	}
}

func isSVGCommand(c byte) bool {
	switch c {
	case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v', 'C', 'c', 'S', 's', 'Q', 'q', 'T', 't', 'A', 'a', 'Z', 'z':
		return true
	}
	return false
}

// svgTokens splits path data into command letters and numbers. Compact forms
// like ".9.9" and "0-.92" split at the next sign or dot, as the SVG grammar
// requires.
func svgTokens(d string) []svgToken {
	var out []svgToken
	for i := 0; i < len(d); {
		c := d[i]
		if isSVGCommand(c) {
			out = append(out, svgToken{cmd: c})
			i++
			continue
		}
		if c == ',' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		start := i
		if c == '+' || c == '-' {
			i++
		}
		dot := false
		for i < len(d) {
			c = d[i]
			if c >= '0' && c <= '9' {
				i++
				continue
			}
			if c == '.' && !dot {
				dot = true
				i++
				continue
			}
			if (c == 'e' || c == 'E') && i+1 < len(d) {
				j := i + 1
				if d[j] == '+' || d[j] == '-' {
					j++
				}
				if j < len(d) && d[j] >= '0' && d[j] <= '9' {
					i = j
					for i < len(d) && d[i] >= '0' && d[i] <= '9' {
						i++
					}
				}
			}
			break
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(d[start:i]), 32); err == nil {
			out = append(out, svgToken{num: float32(v)})
		}
		if i == start {
			i++
		}
	}
	return out
}

// svgArcToCubic converts one elliptical arc (the SVG endpoint form) into up to
// four cubic segments with the same construction browsers use.
func svgArcToCubic(emit func(op byte, args ...float32), from, to [2]float32, rx, ry, rot float32, large, sweep bool) {
	rx, ry = float32(math.Abs(float64(rx))), float32(math.Abs(float64(ry)))
	if rx == 0 || ry == 0 || from == to {
		emit('L', to[0], to[1])
		return
	}
	phi := float64(rot) * math.Pi / 180
	cos, sin := float32(math.Cos(phi)), float32(math.Sin(phi))
	dx, dy := (from[0]-to[0])/2, (from[1]-to[1])/2
	x1 := cos*dx + sin*dy
	y1 := -sin*dx + cos*dy
	lambda := x1*x1/(rx*rx) + y1*y1/(ry*ry)
	if lambda > 1 {
		s := float32(math.Sqrt(float64(lambda)))
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	coef := float32(0)
	if den != 0 && num > 0 {
		coef = float32(math.Sqrt(float64(num / den)))
	}
	if large == sweep {
		coef = -coef
	}
	cx1 := coef * rx * y1 / ry
	cy1 := -coef * ry * x1 / rx
	cx := cos*cx1 - sin*cy1 + (from[0]+to[0])/2
	cy := sin*cx1 + cos*cy1 + (from[1]+to[1])/2
	angle := func(ux, uy, vx, vy float32) float32 {
		lu := float32(math.Hypot(float64(ux), float64(uy)))
		lv := float32(math.Hypot(float64(vx), float64(vy)))
		if lu == 0 || lv == 0 {
			return 0
		}
		dot := (ux*vx + uy*vy) / (lu * lv)
		a := float32(math.Acos(float64(max(-1, min(1, dot)))))
		if ux*vy-uy*vx < 0 {
			a = -a
		}
		return a
	}
	theta := angle(1, 0, (x1-cx1)/rx, (y1-cy1)/ry)
	delta := angle((x1-cx1)/rx, (y1-cy1)/ry, (-x1-cx1)/rx, (-y1-cy1)/ry)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}
	segments := int(math.Ceil(math.Abs(float64(delta)) / (math.Pi / 2)))
	if segments < 1 {
		segments = 1
	}
	step := delta / float32(segments)
	for k := 0; k < segments; k++ {
		a1 := theta + float32(k)*step
		a2 := a1 + step
		alpha := 4.0 / 3.0 * float32(math.Tan(float64(step)/4))
		point := func(a float32) [2]float32 {
			x, y := rx*float32(math.Cos(float64(a))), ry*float32(math.Sin(float64(a)))
			return [2]float32{cos*x - sin*y + cx, sin*x + cos*y + cy}
		}
		deriv := func(a float32) [2]float32 {
			x, y := -rx*float32(math.Sin(float64(a))), ry*float32(math.Cos(float64(a)))
			return [2]float32{cos*x - sin*y, sin*x + cos*y}
		}
		p1, p2 := point(a1), point(a2)
		d1, d2 := deriv(a1), deriv(a2)
		emit('C',
			p1[0]+alpha*d1[0], p1[1]+alpha*d1[1],
			p2[0]-alpha*d2[0], p2[1]-alpha*d2[1],
			p2[0], p2[1])
	}
}
