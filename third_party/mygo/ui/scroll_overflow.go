package ui

// ScrollOverflowX enables horizontal scrolling of content that overflows the
// viewport while keeping ordinary text laid out at the viewport's width.
// Unlike ScrollBoth, it does not give children an unbounded layout width.
func (e Element) ScrollOverflowX() Element {
	if n := e.node(); n != nil {
		n.scrollOverflowX = true
		n.flags |= flagScrollX | flagHover
	}
	return e
}

// overflowRight includes overflowing descendants, stopping at nested clip and
// scroll boundaries whose overflow belongs to their own viewport.
func overflowRight(e *node) float32 {
	right := e.w
	if e.scrolls() || e.flags&flagClipX != 0 {
		return right
	}
	for child := e.first; child != nil; child = child.next {
		if child.flags&flagAbsolute == 0 {
			right = max(right, child.x+overflowRight(child))
		}
	}
	return right
}
