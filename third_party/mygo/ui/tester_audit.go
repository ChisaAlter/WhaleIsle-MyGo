package ui

import "fmt"

// InteractiveBox describes an element of the last frame that takes the
// pointer or the focus, for audits of a view's interaction.
type InteractiveBox struct {
	// Rect is the visible box of the element, in DIPs.
	Rect Rect
	// Label is the text or label of the element, or of the nearest ancestor
	// that has one.
	Label string
	// Role is the element's role for assistive technology.
	Role Role
	// Kinds names what the element takes: click, focus, hover, drag, text.
	Kinds []string
	// Cursor is the pointer shape the window shows over the middle of the
	// element's box, which is what the user sees there.
	Cursor Cursor
}

// Interactives lists the enabled elements of the last frame that take
// clicks, the focus, drags or text, with the pointer shape shown over
// each. It moves the pointer over them, so call it on a view that is
// ready to be looked at, not in the middle of a gesture.
func (t *Tester) Interactives() []InteractiveBox {
	texts := map[uint64]string{}
	for _, n := range t.rt.labels {
		texts[n.id] = n.text
	}
	type entry struct {
		st *state
		r  Rect
	}
	var list []entry
	seen := map[uint64]bool{}
	for _, h := range t.rt.hits {
		s := h.st
		if s == nil || seen[s.id] || h.r.W <= 0 || h.r.H <= 0 {
			continue
		}
		if s.flags&(flagClickable|flagFocusable|flagDraggable|flagEditable) == 0 || s.flags&flagDisabled != 0 {
			continue
		}
		seen[s.id] = true
		list = append(list, entry{s, h.r})
	}
	var out []InteractiveBox
	for _, e := range list {
		s := e.st
		label := texts[s.id]
		for p := s; label == "" && p != nil; p = t.rt.states[p.parent] {
			label = texts[p.id]
			if p.parent == 0 {
				break
			}
		}
		if label == "" {
			// The first text inside the box.
			for _, n := range t.rt.labels {
				if n.r.X >= e.r.X && n.r.Y >= e.r.Y && n.r.X+n.r.W <= e.r.X+e.r.W+1 && n.r.Y+n.r.H <= e.r.Y+e.r.H+1 {
					label = n.text
					break
				}
			}
		}
		var kinds []string
		if s.flags&flagClickable != 0 {
			kinds = append(kinds, "click")
		}
		if s.flags&flagFocusable != 0 {
			kinds = append(kinds, "focus")
		}
		if s.flags&flagDraggable != 0 {
			kinds = append(kinds, "drag")
		}
		if s.flags&flagEditable != 0 {
			kinds = append(kinds, "text")
		}
		t.Move(e.r.X+e.r.W/2, e.r.Y+e.r.H/2)
		out = append(out, InteractiveBox{Rect: e.r, Label: label, Role: s.role, Kinds: kinds, Cursor: t.Cursor()})
	}
	return out
}

// String names the box for logs.
func (b InteractiveBox) String() string {
	return fmt.Sprintf("%v role=%v kinds=%v cursor=%v label=%q", b.Rect, b.Role, b.Kinds, b.Cursor, b.Label)
}
