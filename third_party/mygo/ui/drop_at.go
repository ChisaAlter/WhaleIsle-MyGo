package ui

// DropAt makes e take values of type T dragged within the window, as Drop does,
// and returns the value and its release position in window DIPs. The position
// belongs to the completed drop, even if the pointer moved before this frame.
func DropAt[T any](e Element) (value T, x, y float32, dropped bool) {
	n := e.node()
	if n == nil {
		return
	}
	value, dropped = coreDrop[T](e.nodeFor(n.c.rt))
	if dropped {
		x, y = n.st.dropX, n.st.dropY
	}
	return
}
