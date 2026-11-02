package composer

import (
	"github.com/egoist/mygo/ui"
	"time"
)

const maxDraftUndo = 100

type draftSnapshot struct {
	text       string
	selection  ui.TextInputSelection
	marked     *ui.TextInputRange
	references []draftReference
}

func (c *DraftClient) snapshot() draftSnapshot {
	c.ensure()
	s := draftSnapshot{text: c.lastText, selection: c.selection, references: append([]draftReference(nil), c.references...)}
	if c.marked != nil {
		m := *c.marked
		s.marked = &m
	}
	return s
}

func (c *DraftClient) pushUndo() {
	c.undo = append(c.undo, c.snapshot())
	if len(c.undo) > maxDraftUndo {
		c.undo = c.undo[len(c.undo)-maxDraftUndo:]
	}
	c.redo = c.redo[:0]
}

func (c *DraftClient) restore(s draftSnapshot) {
	if c.Draft != nil {
		*c.Draft = s.text
	}
	if c.lastText != s.text {
		c.revision++
	}
	c.lastText = s.text
	c.references = append(c.references[:0], s.references...)
	c.selection = s.selection
	c.marked = nil
	c.compositionPending = nil
	c.typingUndoAt = time.Time{}
	if s.marked != nil {
		m := *s.marked
		c.marked = &m
	}
	c.shape()
}

// Undo restores the previous document snapshot. Returns false when the stack is empty.
func (c *DraftClient) Undo() bool {
	if len(c.undo) == 0 {
		return false
	}
	c.redo = append(c.redo, c.snapshot())
	s := c.undo[len(c.undo)-1]
	c.undo = c.undo[:len(c.undo)-1]
	c.restore(s)
	return true
}

// Redo reapplies a snapshot undone with Undo.
func (c *DraftClient) Redo() bool {
	if len(c.redo) == 0 {
		return false
	}
	c.undo = append(c.undo, c.snapshot())
	s := c.redo[len(c.redo)-1]
	c.redo = c.redo[:len(c.redo)-1]
	c.restore(s)
	return true
}

// ClearHistory creates a committed document boundary without changing text.
// It belongs to this draft only; ordinary Sync must retain editing history.
func (c *DraftClient) ClearHistory() {
	if c == nil {
		return
	}
	c.Sync()
	c.undo, c.redo = nil, nil
	c.marked = nil
	c.compositionPending = nil
	c.typingUndoAt = time.Time{}
	c.navigationXValid = false
}
