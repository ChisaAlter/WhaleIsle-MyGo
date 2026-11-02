package ui

// HasTextSelection reports whether the current noncollapsed text selection
// intersects this element or a descendant paragraph. Hidden, inert, disabled or
// removed selection owners are excluded. A standalone Selectable text uses its
// focused editor's selection. The query changes no input, focus or selection.
func (e *node) HasTextSelection() bool {
	if !e.hasState() || e.flags&(flagInvisible|flagInert|flagDisabled) != 0 {
		return false
	}
	rt := e.c.rt
	if !rt.selectionStateActive(e.id) {
		return false
	}
	a, b, selected := rt.textSelectionBounds()
	if selected {
		if !rt.selectionStateActive(a.id) || !rt.selectionStateActive(b.id) {
			return false
		}
		first, last := rt.textPointIndex(a), rt.textPointIndex(b)
		for i := first; i <= last; i++ {
			p := rt.texts[i]
			if p.textScope == rt.selection.scope && rt.selectionStateActive(p.id) && rt.within(p.id, e.id) {
				return true
			}
		}
		return false
	}
	p := rt.states[rt.focused]
	if p == nil || p.textScope != 0 || p.flags&flagSelectable == 0 || p.editor == nil || !rt.selectionStateActive(p.id) || !rt.within(p.id, e.id) {
		return false
	}
	for _, current := range rt.texts {
		if current != p {
			continue
		}
		for _, r := range p.editor.selectedRanges() {
			if r.Start != r.End {
				return true
			}
		}
	}
	return false
}

func (rt *engine) selectionStateActive(id uint64) bool {
	for s, n := rt.states[id], 0; s != nil && n < 1024; s, n = rt.states[s.parent], n+1 {
		if s.flags&(flagInvisible|flagInert|flagDisabled) != 0 {
			return false
		}
		if s.parent == 0 {
			return true
		}
	}
	return false
}
