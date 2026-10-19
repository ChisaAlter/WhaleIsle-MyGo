package ui

import "math"

// ObserveLayout observes this frame's unshifted window rectangle before any
// OffsetAfterLayout displacement. It follows the same callback restrictions.
func (e Element) ObserveLayout(observe func(Rect)) Element {
	if n := e.node(); n != nil {
		n.observeLayout = observe
	}
	return e
}

func observeLayout(e *node) {
	if e.observeLayout != nil {
		e.observeLayout(Rect{X: e.x, Y: e.y, W: e.w, H: e.h})
	}
	for child := e.first; child != nil; child = child.next {
		observeLayout(child)
	}
}

// OffsetAfterLayout computes a relative displacement from this frame's laid
// out window rectangle. It moves the element and its children without changing
// sibling flow. Painting, hit testing, accessibility and native text geometry
// all use the displaced rectangle in the same frame.
//
// Children run before parents, so a child may record geometry for an ancestor's
// displacement. The callback runs on the UI thread after layout, before commit;
// it must not build controls or retain Elements from this pass.
func (e Element) OffsetAfterLayout(offset func(Rect) Point) Element {
	if n := e.node(); n != nil {
		n.offsetAfterLayout = offset
	}
	return e
}

func applyLayoutOffsets(e *node) {
	for child := e.first; child != nil; child = child.next {
		applyLayoutOffsets(child)
	}
	if e.offsetAfterLayout == nil {
		return
	}
	d := e.offsetAfterLayout(Rect{X: e.x, Y: e.y, W: e.w, H: e.h})
	if math.IsNaN(float64(d.X)) || math.IsInf(float64(d.X), 0) || math.IsNaN(float64(d.Y)) || math.IsInf(float64(d.Y), 0) {
		return
	}
	if d.X != 0 || d.Y != 0 {
		shiftLayoutSubtree(e, d.X, d.Y)
	}
}

func shiftLayoutSubtree(e *node, dx, dy float32) {
	e.x += dx
	e.y += dy
	for child := e.first; child != nil; child = child.next {
		shiftLayoutSubtree(child, dx, dy)
	}
}
