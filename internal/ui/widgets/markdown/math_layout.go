package markdown

import "github.com/egoist/mygo/ui"

type mathAtom struct {
	width, ascent, descent float32
	text                   *ui.TextLayout
	svg                    *ui.SVG
	color                  ui.Color
	label, target          string
	newline, display       bool
}

type mathPlacement struct {
	atom mathAtom
	rect ui.Rect
}

type mathLayout struct {
	items  []mathPlacement
	height float32
}

// layoutMathAtoms performs genuine baseline alignment: a fraction's entire
// numerator sits above the shared baseline and its denominator below it.
// Display formulas take a separate centered line; inline formulas are atomic
// and wrap with adjacent text. Oversized formula images scale to the column.
func layoutMathAtoms(atoms []mathAtom, width, lineHeight float32) mathLayout {
	width = max(1, width)
	var result mathLayout
	var line []mathAtom
	var used, ascent, descent, y float32
	flush := func(force bool) {
		if len(line) == 0 {
			if force {
				y += lineHeight
			}
			return
		}
		x := float32(0)
		for _, atom := range line {
			if atom.display {
				x = max(0, (width-atom.width)/2)
			}
			result.items = append(result.items, mathPlacement{atom: atom, rect: ui.Rect{X: x, Y: y + ascent - atom.ascent, W: atom.width, H: atom.ascent + atom.descent}})
			x += atom.width
		}
		y += max(lineHeight, ascent+descent)
		line, used, ascent, descent = nil, 0, 0, 0
	}
	for _, atom := range atoms {
		if atom.newline {
			flush(true)
			continue
		}
		if atom.svg != nil && atom.width > width {
			scale := width / atom.width
			atom.width, atom.ascent, atom.descent = width, atom.ascent*scale, atom.descent*scale
		}
		if atom.display {
			flush(false)
		}
		if used > 0 && used+atom.width > width {
			flush(false)
		}
		line = append(line, atom)
		used += atom.width
		ascent, descent = max(ascent, atom.ascent), max(descent, atom.descent)
		if atom.display {
			flush(false)
		}
	}
	flush(false)
	result.height = max(lineHeight, y)
	return result
}
