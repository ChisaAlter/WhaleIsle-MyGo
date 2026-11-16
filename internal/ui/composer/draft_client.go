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

// InputBar.input uses 4px 8px 0 14px padding in both modes; Hero only raises
// its minimum height. The card's 8px top padding is outside this writing well.
func draftInsets(_ bool) (left, right, top float32) {
	return draftInputLeft, draftInputRight, draftInputTop
}

// setInsets records the frame's inner insets so native caret and hit-test
// geometry agree with the painted text in both card layouts.
func (c *DraftClient) setInsets(left, right, top float32) {
	c.insetLeft, c.insetRight, c.insetTop, c.insetSet = left, right, top, true
}

// ins returns the insets in effect for this frame.
func (c *DraftClient) ins() (left, right, top float32) {
	if c.insetSet {
		return c.insetLeft, c.insetRight, c.insetTop
	}
	return draftInputLeft, draftInputRight, draftInputTop
}

// DraftClient implements ui.TextInputClient for the composer draft (*string).
// Selection and IME marked ranges use UTF-16 code units, matching MyGo native
// text services. Call Sync after programmatic draft edits from chips or suggest.
type DraftClient struct {
	insetLeft, insetRight, insetTop float32
	insetSet                        bool
	regionStop                      [2]stopSequence
	suggestions                     SuggestionState
	stop                            stopSequence
	// Native text input retains this adapter identity across frames so an
	// ordinary repaint does not release an active IME composition.
	input              typingInputClient
	altInput           typingInputClient
	useAltInput        bool
	accessPlaceholder  string
	typing             typingEffects
	Draft              *string
	selection          ui.TextInputSelection
	marked             *ui.TextInputRange
	compositionPending *ui.TextInputRange
	imeClosingEnter    bool
	imeClosingEscape   bool
	imeProcessKey      string
	imeProcessActive   bool
	typingUndoAt       time.Time
	typingUndoEnd      int
	layout             *ui.TextLayout // display projection, not the clipboard string
	displayText        string
	references         []draftReference
	inlineChips        []draftInlineChip
	nativeTyping       bool
	readClipboard      func() string
	writeClipboard     func(string)
	pointerDown        bool
	pointerStart       ui.Point
	pointerChip        *draftReference
	pointerRemove      bool
	pointerDragging    bool
	lastText           string
	revision           uint64
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
	c.altInput.DraftClient = c
	if draft != nil {
		c.lastText = *draft
		end := ui.UTF16Len(c.lastText)
		c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}
		c.references = draftReferences(c.lastText)
	}
	c.shape()
	return c
}

func (c *DraftClient) textInputHandle() ui.TextInputClient {
	c.input.DraftClient = c
	c.altInput.DraftClient = c
	if c.useAltInput {
		return &c.altInput
	}
	return &c.input
}

func isEnterPhysicalCode(code string) bool {
	return code == "Enter" || code == "NumpadEnter"
}

func (c *DraftClient) clearIMEKeyLatch() {
	c.imeClosingEnter = false
	c.imeClosingEscape = false
	c.imeProcessKey = ""
	c.imeProcessActive = false
}

func (c *DraftClient) cancelComposition() {
	if c.marked != nil || c.compositionPending != nil || c.typing.composing || c.imeClosingEnter || c.imeClosingEscape || c.imeProcessActive {
		c.useAltInput = !c.useAltInput
	}
	c.marked = nil
	c.compositionPending = nil
	c.typing.composing = false
	c.clearIMEKeyLatch()
}

func (c *DraftClient) noteIMEProcessKey(code string) {
	c.imeProcessKey = code
	c.imeProcessActive = true
	switch {
	case code == "" || isEnterPhysicalCode(code):
		c.imeClosingEnter = true
		c.imeClosingEscape = false
	case code == "Escape":
		c.imeClosingEscape = true
		c.imeClosingEnter = false
	default:
		c.imeClosingEnter = false
		c.imeClosingEscape = false
	}
}

func (c *DraftClient) noteIMEKeyUp(key ui.Key, code string) {
	if key == ui.KeyEnter || isEnterPhysicalCode(code) {
		c.imeClosingEnter = false
		if isEnterPhysicalCode(c.imeProcessKey) || c.imeProcessKey == "" {
			c.imeProcessActive = false
			c.imeProcessKey = ""
		}
		return
	}
	if key == ui.KeyEscape || code == "Escape" {
		c.imeClosingEscape = false
		if c.imeProcessKey == "Escape" || c.imeProcessKey == "" {
			c.imeProcessActive = false
			c.imeProcessKey = ""
		}
		return
	}
	if c.imeProcessActive && (code == "" || code == c.imeProcessKey) {
		c.imeProcessActive = false
		c.imeProcessKey = ""
	}
}

func (c *DraftClient) consumeClosingEnter() bool {
	if c.marked != nil {
		return true
	}
	if c.imeClosingEnter || (c.imeProcessActive && (isEnterPhysicalCode(c.imeProcessKey) || c.imeProcessKey == "")) {
		c.imeClosingEnter = false
		if isEnterPhysicalCode(c.imeProcessKey) || c.imeProcessKey == "" {
			c.imeProcessActive = false
			c.imeProcessKey = ""
		}
		return true
	}
	return false
}

func (c *DraftClient) consumeClosingEscape() bool {
	if c.marked != nil {
		return true
	}
	if c.imeClosingEscape || (c.imeProcessActive && (c.imeProcessKey == "Escape" || c.imeProcessKey == "")) {
		c.imeClosingEscape = false
		if c.imeProcessKey == "Escape" || c.imeProcessKey == "" {
			c.imeProcessActive = false
			c.imeProcessKey = ""
		}
		return true
	}
	return false
}

// Sync reconciles selection after the draft was changed without ReplaceText.
func (c *DraftClient) Sync() {
	if c.font.Family == "" {
		c.syncFont(theme.RenderPreferences().Composer)
		return
	}
	c.ensure()
}

// syncFont uses this frame's derived skin role for shaping, caret and hit tests.
// The durable preference and the retained native text-input identity stay intact.
func (c *DraftClient) syncFont(p theme.FontPreference) {
	if c.font.Family != p.Family || c.font.Size != p.Size {
		c.font.Family = p.Family
		c.font.Size = p.Size
		c.shape()
	}
	c.ensure()
}

// TextInputAccessibility shares the same document and undo owner with native
// input while letting assistive technology read and replace the whole draft.
func (c *DraftClient) TextInputAccessibility() ui.TextInputAccessibility {
	return ui.TextInputAccessibility{Multiline: true, Placeholder: c.accessPlaceholder}
}

// SetSelectionUTF16 sets the primary selection in UTF-16 document offsets.
func (c *DraftClient) SetSelectionUTF16(start, end int) {
	c.ensure()
	c.typingUndoAt = time.Time{}
	c.selection = ui.TextInputSelection{Range: c.normalizeRange(ui.TextInputRange{Start: start, End: end}), Reversed: start > end}
	c.cancelComposition()
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
		c.revision++
		c.lastText = cur
		c.references = draftReferences(cur)
		c.cancelComposition()
		c.typingUndoAt = time.Time{}
		c.clampSelection()
		c.shape()
	}
}

func (c *DraftClient) clampSelection() {
	n := ui.UTF16Len(c.lastText)
	c.selection.Range = c.normalizeRange(c.selection.Range)
	if c.marked != nil {
		m := *c.marked
		m.Start = max(0, min(m.Start, n))
		m.End = max(0, min(m.End, n))
		c.marked = &m
	}
}

func (c *DraftClient) shape() {
	c.navigationXValid = false
	c.buildProjection()
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

func (c *DraftClient) replace(r ui.TextInputRange, text string, importText bool) {
	c.ensure()
	s := c.text()
	a, z := ui.UTF16ByteOffset(s, r.Start), ui.UTF16ByteOffset(s, r.End)
	next := s[:a] + text + s[z:]
	if c.Draft != nil {
		*c.Draft = next
	}
	if c.lastText != next {
		c.revision++
	}
	c.lastText = next
	c.editReferences(r, text, importText, c.nativeTyping)
	end := r.Start + ui.UTF16Len(text)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: end, End: end}}
	c.shape()
}

func (c *DraftClient) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	c.ensure()
	s := c.text()
	r = c.normalizeRange(r)
	a, z := ui.UTF16ByteOffset(s, r.Start), ui.UTF16ByteOffset(s, r.End)
	return s[a:z], r
}

func (c *DraftClient) Selection() ui.TextInputSelection {
	c.ensure()
	c.clampSelection()
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
	wanted := c.normalizeRange(c.target(r))
	typing := !committing && r == nil && wanted.Start == wanted.End && text != "\n" && utf8.RuneCountInString(text) == 1
	now := time.Now()
	merge := typing && !c.typingUndoAt.IsZero() && now.Sub(c.typingUndoAt) < time.Second && c.typingUndoEnd == c.selection.Caret()
	if !committing && !merge {
		c.pushUndo()
	}
	c.replace(wanted, text, true)
	c.marked = nil
	c.compositionPending = nil
	if !committing {
		c.clearIMEKeyLatch()
	} else {
		c.imeClosingEscape = false
		if !c.imeProcessActive || (!isEnterPhysicalCode(c.imeProcessKey) && c.imeProcessKey != "") {
			c.imeClosingEnter = false
			c.imeProcessActive = false
			c.imeProcessKey = ""
		}
	}
	c.typingUndoAt = time.Time{}
	if typing {
		c.typingUndoAt, c.typingUndoEnd = now, c.selection.Caret()
	}
}

func (c *DraftClient) SetMarkedText(r *ui.TextInputRange, text string, selected ui.TextInputRange) {
	c.stop.reset()
	c.ensure()
	c.typingUndoAt = time.Time{}
	if c.marked == nil {
		c.pushUndo()
	}
	c.compositionPending = nil
	c.imeClosingEnter = false
	c.imeClosingEscape = false
	wanted := c.normalizeRange(c.target(r))
	c.replace(wanted, text, false)
	mark := ui.TextInputRange{Start: wanted.Start, End: wanted.Start + ui.UTF16Len(text)}
	c.marked = &mark
	length := ui.UTF16Len(text)
	selected.Start, selected.End = max(0, min(selected.Start, length)), max(0, min(selected.End, length))
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: mark.Start + min(selected.Start, selected.End), End: mark.Start + max(selected.Start, selected.End)}, Reversed: selected.Start > selected.End}
}

func (c *DraftClient) UnmarkText() {
	if c.marked != nil {
		r := *c.marked
		c.compositionPending = &r
		if !c.imeProcessActive || isEnterPhysicalCode(c.imeProcessKey) || c.imeProcessKey == "" {
			c.imeClosingEnter = true
		}
		if c.imeProcessActive && c.imeProcessKey == "Escape" {
			c.imeClosingEscape = true
		}
	}
	c.marked = nil
}

func (c *DraftClient) composing() bool {
	return c.marked != nil || c.imeClosingEnter || c.imeClosingEscape || c.imeProcessActive
}

func (c *DraftClient) BoundsForRange(r ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	c.ensure()
	if c.layout == nil {
		return ui.Rect{}, r, false
	}
	r = c.normalizeRange(r)
	b := c.visualCaret(r.Start)
	if r.Start != r.End {
		rects := c.visualSelectionRects(r)
		if len(rects) > 0 {
			b = rects[0]
		}
	}
	l, _, t := c.ins()
	b.X += l
	b.Y += t - c.scrollY
	return b, r, true
}

func (c *DraftClient) IndexForPoint(p ui.Point) (int, bool) {
	c.ensure()
	if c.layout == nil {
		return 0, false
	}
	left, _, top := c.ins()
	return c.visualIndex(ui.Point{X: p.X - left, Y: p.Y - top + c.scrollY}), true
}

func (c *DraftClient) move(to int, extend bool) {
	c.navigationXValid = false
	c.typingUndoAt = time.Time{}
	c.ensure()
	n := ui.UTF16Len(c.text())
	to = max(0, min(to, n))
	to = c.normalizeRange(ui.TextInputRange{Start: to, End: to}).Start
	c.revealCaret = true
	if !extend {
		c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: to, End: to}}
		c.revealCaretInViewport()
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
	_, _, t := c.ins()
	limit := max(0, c.visualHeight()+t-c.viewportHeight)
	caret := c.visualCaret(c.selection.Caret())
	topY, bottom := caret.Y+t, caret.Y+t+caret.H
	if topY < c.scrollY {
		c.scrollY = max(0, topY-t)
	}
	if bottom > c.scrollY+c.viewportHeight {
		c.scrollY = min(limit, bottom-c.viewportHeight)
	}
	c.revealCaret = false
}

// Revision is the mutation identity used by deferred command span consumption.
func (c *DraftClient) Revision() uint64 { c.ensure(); return c.revision }
