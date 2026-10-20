package composer

import (
	"sort"

	"github.com/egoist/mygo/ui"
)

type draftVisualLine struct{ rawY, rawHeight float32 }

// MyGo's exported Font does not expose line height. Preserve its retained
// glyph/grapheme shaping for native hit testing and project only vertical
// geometry; the display Text uses the same 14px font with FixedLineHeight(24).
func (c *DraftClient) buildVisualLines() {
	c.visualLines = c.visualLines[:0]
	seen := make(map[float32]struct{})
	end := ui.UTF16Len(c.lastText)
	for at := 0; ; {
		b := c.layout.Caret(at)
		if _, found := seen[b.Y]; !found {
			seen[b.Y] = struct{}{}
			c.visualLines = append(c.visualLines, draftVisualLine{b.Y, b.H})
		}
		if at >= end {
			break
		}
		next := c.layout.NextBoundary(at)
		if next <= at {
			break
		}
		at = next
	}
	sort.Slice(c.visualLines, func(i, j int) bool { return c.visualLines[i].rawY < c.visualLines[j].rawY })
}

func (c *DraftClient) projectRect(b ui.Rect) ui.Rect {
	line := max(0, sort.Search(len(c.visualLines), func(i int) bool { return c.visualLines[i].rawY > b.Y+.01 })-1)
	b.Y = float32(line) * c.lineHeight()
	b.H = c.lineHeight()
	return b
}
func (c *DraftClient) visualCaret(at int) ui.Rect { return c.projectRect(c.layout.Caret(at)) }
func (c *DraftClient) visualSelectionRects(r ui.TextInputRange) []ui.Rect {
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
