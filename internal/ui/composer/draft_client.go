package composer

import (
	"github.com/egoist/mygo/ui"
	"time"
	"unicode/utf8"
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
	regionStop  [2]stopSequence
	suggestions SuggestionState
	stop        stopSequence
	// Native text input retains this adapter identity across frames so an
	// ordinary repaint does not release an active IME composition.
	input              typingInputClient
	typing             typingEffects
	Draft              *string
	selection          ui.TextInputSelection
	marked             *ui.TextInputRange
	compositionPending *ui.TextInputRange
	compositionEnded   time.Time
	typingUndoAt       time.Time
	typingUndoEnd      int
	layout             *ui.TextLayout
	lastText           string
	wrapWidth          float32
	font               ui.Font
	// Display geometry is a fixed24px projection of the toolkit's retained
	// shaping. The native text owner, UTF16 offsets, and edit history stay here.
	visualLines      []draftVisualLine
	scrollY          float32
	viewportHeight   float32
	revealCaret      bool
	navigationX      float32
	navigationXValid bool
	undo             []draftSnapshot
	redo             []draftSnapshot
}

// NewDraftClient returns a client bound to draft. The pointer must stay stable.
func NewDraftClient(draft *string) *DraftClient {
	c := &DraftClient{Draft: draft, font: ui.Font{Size: 14}}
	c.input.DraftClient = c
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
	p := theme.RenderPreferences().Composer
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
	c.typingUndoAt = time.Time{}
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: start, End: end}}
	c.marked = nil
	c.revealCaret = true
	c.revealCaretInViewport()
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
		c.compositionPending = nil
		c.typingUndoAt = time.Time{}
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
	c.navigationXValid = false
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
	c.stop.reset()
	c.ensure()
	// A native IME commit replaces its marked range. The pre-composition
	// snapshot already owns this edit; intermediate candidates are not undo
	// steps and must never be restored as an active composition.
	committing := c.marked != nil || c.compositionPending != nil && r != nil && *r == *c.compositionPending
	typing := !committing && r == nil && c.selection.Range.Start == c.selection.Range.End && text != "\n" && utf8.RuneCountInString(text) == 1
	now := time.Now()
	merge := typing && !c.typingUndoAt.IsZero() && now.Sub(c.typingUndoAt) < time.Second && c.typingUndoEnd == c.selection.Caret()
	if !committing && !merge {
		c.pushUndo()
	}
	c.replace(c.target(r), text)
	c.marked = nil
	c.compositionPending = nil
	c.typingUndoAt = time.Time{}
	if typing {
		c.typingUndoAt, c.typingUndoEnd = now, c.selection.Caret()
	}
}

func (c *DraftClient) SetMarkedText(r *ui.TextInputRange, text string, selected ui.TextInputRange) {
	c.stop.reset()
	c.typingUndoAt = time.Time{}
	if c.marked == nil {
		c.pushUndo()
	}
	c.compositionPending = nil
	wanted := c.target(r)
	c.replace(wanted, text)
	mark := ui.TextInputRange{Start: wanted.Start, End: wanted.Start + ui.UTF16Len(text)}
	c.marked = &mark
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: mark.Start + selected.Start, End: mark.Start + selected.End}}
}

func (c *DraftClient) UnmarkText() {
	if c.marked != nil {
		r := *c.marked
		c.compositionPending = &r
		c.compositionEnded = time.Now()
	}
	c.marked = nil
}

func (c *DraftClient) composing() bool {
	return c.marked != nil || c.compositionPending != nil && time.Since(c.compositionEnded) < 10*time.Millisecond
}

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
	c.navigationXValid = false
	c.typingUndoAt = time.Time{}
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
	c.revealCaretInViewport()
}

// revealCaretInViewport keeps keyboard navigation immediately visible. The UI
// layout pass also calls setViewport, but native key events can be observed by
// callers before the next frame is painted.
func (c *DraftClient) revealCaretInViewport() {
	if c.layout == nil || c.viewportHeight <= 0 {
		return
	}
	limit := max(0, c.visualHeight()+draftInputTop-c.viewportHeight)
	caret := c.visualCaret(c.selection.Caret())
	top, bottom := caret.Y+draftInputTop, caret.Y+draftInputTop+caret.H
	if top < c.scrollY {
		c.scrollY = max(0, top-draftInputTop)
	}
	if bottom > c.scrollY+c.viewportHeight {
		c.scrollY = min(limit, bottom-c.viewportHeight)
	}
	c.revealCaret = false
}
