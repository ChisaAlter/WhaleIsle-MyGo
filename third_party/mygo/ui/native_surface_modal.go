package ui

// HasForegroundModal lets native child surfaces honor the same modal ownership
// as canvas widgets. Call after building the view, before presenting the frame.
func (c *Context) HasForegroundModal() bool {
	raw := c.build()
	if raw == nil {
		return true
	}
	var contains func(*node) bool
	contains = func(n *node) bool {
		for ; n != nil; n = n.next {
			if n.flags&flagModal != 0 || contains(n.first) {
				return true
			}
		}
		return false
	}
	return contains(raw.overlay)
}
