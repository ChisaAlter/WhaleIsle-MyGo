package ui

import "time"

// LastInputTime returns the latest pointer, keyboard, composition or focus event
// for this window. Paint, layout and background data updates never advance it.
// The zero value means the window has not received user input yet.
func (c *Context) LastInputTime() time.Time {
	if raw := c.build(); raw != nil {
		return raw.rt.lastInputAt
	}
	return time.Time{}
}
