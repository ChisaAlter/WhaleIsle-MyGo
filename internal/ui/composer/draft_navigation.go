package composer

import (
	"runtime"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Navigation runs after suggestion arbitration and yields to native IME.
func (c *DraftClient) navigate(ev ui.InputEvent) bool {
	if c.marked != nil {
		return false
	}
	switch ev.Key {
	case ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown, ui.KeyHome, ui.KeyEnd:
	default:
		return false
	}
	c.ensure()
	mods := ev.Mods &^ ui.Shift
	word := mods == ui.Ctrl
	line := false
	if runtime.GOOS == "darwin" {
		word = mods == ui.Alt
		line = mods == ui.Super
	}
	if mods != 0 && !word && !line {
		return false
	}
	if word && (ev.Key == ui.KeyUp || ev.Key == ui.KeyDown) {
		return false
	}
	extend := ev.Mods&ui.Shift != 0
	at := c.selection.Caret()
	to := at
	vertical := false
	x := float32(0)
	switch ev.Key {
	case ui.KeyLeft, ui.KeyRight:
		forward := ev.Key == ui.KeyRight
		if !extend && c.selection.Range.Start != c.selection.Range.End && mods == 0 {
			to = c.selection.Range.Start
			if forward {
				to = c.selection.Range.End
			}
		} else if word {
			to = c.wordBoundary(at, forward)
		} else if line {
			b := c.visualCaret(at)
			x = -1
			if forward {
				x = 1e7
			}
			to = c.layout.IndexAt(c.rawPoint(ui.Point{X: x, Y: b.Y + b.H/2}))
		} else if forward {
			to = c.layout.NextBoundary(at)
		} else {
			to = c.layout.PreviousBoundary(at)
		}
	case ui.KeyHome, ui.KeyEnd:
		if mods == ui.Ctrl || line {
			to = 0
			if ev.Key == ui.KeyEnd {
				to = ui.UTF16Len(c.text())
			}
		} else {
			b := c.visualCaret(at)
			x = -1
			if ev.Key == ui.KeyEnd {
				x = 1e7
			}
			to = c.layout.IndexAt(c.rawPoint(ui.Point{X: x, Y: b.Y + b.H/2}))
		}
	case ui.KeyUp, ui.KeyDown:
		if line {
			to = 0
			if ev.Key == ui.KeyDown {
				to = ui.UTF16Len(c.text())
			}
			break
		}
		b := c.visualCaret(at)
		x = b.X
		if c.navigationXValid {
			x = c.navigationX
		}
		y := b.Y - c.lineHeight()
		if ev.Key == ui.KeyDown {
			y = b.Y + c.lineHeight()
		}
		if y < 0 {
			to = 0
		} else if y >= c.visualHeight() {
			to = ui.UTF16Len(c.text())
		} else {
			to = c.layout.IndexAt(c.rawPoint(ui.Point{X: x, Y: y + b.H/2}))
		}
		vertical = true
	}
	c.move(to, extend)
	if vertical {
		c.navigationX, c.navigationXValid = x, true
	}
	return true
}

// Skip separators then one word when going back; consume one word then its
// trailing separators when going forward. All destinations remain grapheme-safe.
func (c *DraftClient) wordBoundary(at int, forward bool) int {
	text := c.text()
	isWord := func(at int) bool {
		b := ui.UTF16ByteOffset(text, at)
		if b >= len(text) {
			return false
		}
		r, _ := utf8.DecodeRuneInString(text[b:])
		return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) || r == '_'
	}
	end := ui.UTF16Len(text)
	if forward {
		for at < end && isWord(at) {
			at = c.layout.NextBoundary(at)
		}
		for at < end && !isWord(at) {
			at = c.layout.NextBoundary(at)
		}
	} else {
		for at > 0 {
			p := c.layout.PreviousBoundary(at)
			if isWord(p) {
				break
			}
			at = p
		}
		for at > 0 {
			p := c.layout.PreviousBoundary(at)
			if !isWord(p) {
				break
			}
			at = p
		}
	}
	return at
}
