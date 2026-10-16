package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

const (
	draftInputLeft       = float32(14)
	draftInputRight      = float32(8)
	draftInputTop        = float32(4)
	draftInputLineHeight = float32(24)
	draftInputMaxHeight  = float32(336)
)

// DraftClient implements ui.TextInputClient for the composer draft (*string).
// Selection and IME marked ranges use UTF-16 code units, matching MyGo native
// text services. Call Sync after programmatic draft edits from chips or suggest.
type DraftClient struct {
	typing    typingEffects
	Draft     *string
	selection ui.TextInputSelection
	marked    *ui.TextInputRange
	layout    *ui.TextLayout
	lastText  string
	wrapWidth float32
	font      ui.Font
	// Display geometry is a fixed24px projection of the toolkit's retained
	// shaping. The native text owner, UTF16 offsets, and edit history stay here.
	visualLines    []draftVisualLine
	scrollY        float32
	viewportHeight float32
	revealCaret    bool
	undo           []draftSnapshot
	redo           []draftSnapshot
}

// NewDraftClient returns a client bound to draft. The pointer must stay stable.
func NewDraftClient(draft *string) *DraftClient {
	c := &DraftClient{Draft: draft, font: ui.Font{Size: 14}}
	if draft != nil {
		c.lastText = *draft
		end := ui.UTF16Len(c.lastText)
		c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}
	}
	c.shape()
	return c
}

// Sync reconciles selection after the draft was changed without ReplaceText.
func (c *DraftClient) Sync() {
	p := theme.CurrentPreferences().Composer
	if c.font.Family != p.Family || c.font.Size != p.Size {
		c.font.Family = p.Family
		c.font.Size = p.Size
		c.shape()
	}
	c.ensure()
}

// SetSelectionUTF16 sets the primary selection in UTF-16 document offsets.
func (c *DraftClient) SetSelectionUTF16(start, end int) {
	c.ensure()
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: start, End: end}}
	c.marked = nil
	c.revealCaret = true
}

func (c *DraftClient) text() string {
	if c.Draft == nil {
		return ""
	}
	return *c.Draft
}

func (c *DraftClient) ensure() {
	if c.Draft == nil {
		return
	}
	cur := *c.Draft
	if cur != c.lastText {
		c.lastText = cur
		c.marked = nil
		c.clampSelection()
		c.shape()
	}
}

func (c *DraftClient) clampSelection() {
	n := ui.UTF16Len(c.lastText)
	r := c.selection.Range
	r.Start = max(0, min(r.Start, n))
	r.End = max(0, min(r.End, n))
	c.selection.Range = r
	if c.marked != nil {
		m := *c.marked
		m.Start = max(0, min(m.Start, n))
		m.End = max(0, min(m.End, n))
		c.marked = &m
	}
}

func (c *DraftClient) shape() {
	c.layout = ui.ShapeText(c.lastText, c.font, c.wrapWidth)
	c.buildVisualLines()
	c.revealCaret = true
}

func (c *DraftClient) setWrapWidth(w float32) {
	if w < 1 {
		w = 0
	}
	if w == c.wrapWidth {
		return
	}
	c.wrapWidth = w
	c.shape()
}

func (c *DraftClient) target(r *ui.TextInputRange) ui.TextInputRange {
	if r != nil {
		return *r
	}
	if c.marked != nil {
		return *c.marked
	}
	return c.selection.Range
}

func (c *DraftClient) replace(r ui.TextInputRange, text string) {
	c.ensure()
	s := c.text()
	a, z := ui.UTF16ByteOffset(s, r.Start), ui.UTF16ByteOffset(s, r.End)
	next := s[:a] + text + s[z:]
	if c.Draft != nil {
		*c.Draft = next
	}
	c.lastText = next
	end := r.Start + ui.UTF16Len(text)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}
	c.shape()
}

func (c *DraftClient) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	c.ensure()
	s := c.text()
	a, z := ui.UTF16ByteOffset(s, r.Start), ui.UTF16ByteOffset(s, r.End)
	return s[a:z], ui.TextInputRange{Start: ui.UTF16Len(s[:a]), End: ui.UTF16Len(s[:z])}
}

func (c *DraftClient) Selection() ui.TextInputSelection {
	c.ensure()
	return c.selection
}

func (c *DraftClient) MarkedRange() (ui.TextInputRange, bool) {
	c.ensure()
	if c.marked != nil {
		return *c.marked, true
	}
	return ui.TextInputRange{}, false
}

func (c *DraftClient) ReplaceText(r *ui.TextInputRange, text string) {
	c.pushUndo()
	c.replace(c.target(r), text)
	c.marked = nil
}

func (c *DraftClient) SetMarkedText(r *ui.TextInputRange, text string, selected ui.TextInputRange) {
	c.pushUndo()
	wanted := c.target(r)
	c.replace(wanted, text)
	mark := ui.TextInputRange{Start: wanted.Start, End: wanted.Start + ui.UTF16Len(text)}
	c.marked = &mark
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: mark.Start + selected.Start, End: mark.Start + selected.End}}
}

func (c *DraftClient) UnmarkText() { c.marked = nil }

func (c *DraftClient) BoundsForRange(r ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	c.ensure()
	if c.layout == nil {
		return ui.Rect{}, r, false
	}
	b := c.visualCaret(r.Start)
	if r.Start != r.End {
		rects := c.visualSelectionRects(r)
		if len(rects) > 0 {
			b = rects[0]
		}
	}
	b.X += draftInputLeft
	b.Y += draftInputTop - c.scrollY
	return b, r, true
}

func (c *DraftClient) IndexForPoint(p ui.Point) (int, bool) {
	c.ensure()
	if c.layout == nil {
		return 0, false
	}
	return c.layout.IndexAt(c.rawPoint(ui.Point{X: p.X - draftInputLeft, Y: p.Y - draftInputTop + c.scrollY})), true
}

func (c *DraftClient) move(to int, extend bool) {
	c.ensure()
	n := ui.UTF16Len(c.text())
	to = max(0, min(to, n))
	c.revealCaret = true
	if !extend {
		c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: to, End: to}}
		return
	}
	anchor := c.selection.Range.Start
	if c.selection.Reversed {
		anchor = c.selection.Range.End
	}
	c.selection = ui.TextInputSelection{
		Range:    ui.TextInputRange{Start: min(anchor, to), End: max(anchor, to)},
		Reversed: to < anchor,
	}
}
