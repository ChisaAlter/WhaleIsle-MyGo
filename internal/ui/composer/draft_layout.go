package composer

import (
	"sort"

	"github.com/egoist/mygo/ui"
)

type draftVisualLine struct{ rawY, rawHeight float32 }

// MyGo's exported Font does not expose line height. Preserve its retained
// glyph/grapheme shaping for native hit testing and project only vertical
// geometry; the display Text uses the same 14px font with FixedLineHeight(24).
//
// Caret Y is monotonic along displayText, so single-line drafts resolve in O(1)
// (two Caret calls, zero allocations) and multi-line drafts advance line by
// line via binary search in O(lines * log(N)) instead of scanning every
// grapheme and allocating a map.
func (c *DraftClient) buildVisualLines() {
	c.visualLines = c.visualLines[:0]
	end := ui.UTF16Len(c.displayText)
	first := c.layout.Caret(0)
	c.visualLines = append(c.visualLines, draftVisualLine{first.Y, first.H})
	if end <= 0 {
		return
	}
	last := c.layout.Caret(end)
	if last.Y <= first.Y+0.01 {
		return
	}
	// Multi-line: jump to the start of each subsequent visual line via binary
	// search over UTF-16 offsets instead of visiting every grapheme.
	lo := 0
	curY := first.Y
	for lo < end {
		// Find smallest offset in (lo, end] whose Caret.Y > curY + 0.01.
		l, r := lo+1, end
		for l < r {
			mid := int(uint(l+r) >> 1)
			if c.layout.Caret(mid).Y > curY+0.01 {
				r = mid
			} else {
				l = mid + 1
			}
		}
		b := c.layout.Caret(l)
		if b.Y <= curY+0.01 {
			break
		}
		c.visualLines = append(c.visualLines, draftVisualLine{b.Y, b.H})
		curY = b.Y
		lo = l
		if curY >= last.Y-0.01 {
			break
		}
	}
}

func (c *DraftClient) projectRect(b ui.Rect) ui.Rect {
	line := max(0, sort.Search(len(c.visualLines), func(i int) bool { return c.visualLines[i].rawY > b.Y+.01 })-1)
	b.Y = float32(line) * c.lineHeight()
	b.H = c.lineHeight()
	return b
}
func (c *DraftClient) visualCaret(at int) ui.Rect {
	return c.projectRect(c.layout.Caret(c.documentToDisplay(at, true)))
}
func (c *DraftClient) visualSelectionRects(r ui.TextInputRange) []ui.Rect {
	if r.Start == r.End {
		return nil
	}
	r = c.normalizeRange(r)
	r.Start, r.End = c.documentToDisplay(r.Start, false), c.documentToDisplay(r.End, true)
	rects := c.layout.SelectionRects(r)
	for i := range rects {
		rects[i] = c.projectRect(rects[i])
	}
	return rects
}
func (c *DraftClient) rawPoint(p ui.Point) ui.Point {
	if len(c.visualLines) == 0 {
		return p
	}
	line := min(len(c.visualLines)-1, max(0, int(p.Y/c.lineHeight())))
	v := c.visualLines[line]
	p.Y = v.rawY + v.rawHeight/2
	return p
}
func (c *DraftClient) visualHeight() float32 {
	return float32(len(c.visualLines)) * c.lineHeight()
}

func (c *DraftClient) setViewport(height float32) {
	c.viewportHeight = height
	_, _, t := c.ins()
	limit := max(0, c.visualHeight()+t-height)
	c.scrollY = min(limit, max(0, c.scrollY))
	if c.revealCaret {
		c.revealCaretInViewport()
	}
}
func (c *DraftClient) scrollBy(dy float32) bool {
	_, _, t := c.ins()
	limit := max(0, c.visualHeight()+t-c.viewportHeight)
	if limit == 0 {
		return false
	}
	previous := c.scrollY
	c.scrollY = min(limit, max(0, c.scrollY+dy))
	c.revealCaret = false
	return c.scrollY != previous
}
