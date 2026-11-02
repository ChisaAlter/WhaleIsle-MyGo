package ui

// Inert keeps the element and its descendants painted in their layout while
// excluding them from pointer hits, keyboard focus, semantic test labels and
// the accessibility tree. Use it for content retained during an exit animation.
// Disabled is separate: combine it with Inert when retained controls must also
// reject pending input and cached accessibility actions.
func (e *node) Inert(on bool) *node {
	if on {
		e.flags |= flagInert
	} else {
		e.flags &^= flagInert
	}
	return e
}
