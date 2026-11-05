package ui

// CloseTopDialog requests the same dismissal as Escape for the currently
// focused top DialogBase. The live owner consumes it during its next build,
// so its existing cancellation, saving and draft policy remain authoritative.
// Unknown modal surfaces are not dismissed and never fall through to a parent.
func (c *Context) CloseTopDialog() bool {
	raw := c.build()
	if raw == nil || raw.inert {
		return false
	}
	rt := raw.rt
	owner := rt.states[rt.modal]
	if rt.modal == 0 || owner == nil || owner.dialogBaseFrame == 0 || owner.dialogBaseFrame+1 < rt.frame {
		return false
	}
	for _, scope := range rt.focusScopes {
		if scope.anchor != 0 {
			return false
		}
	}
	rt.dialogCloseRequest = rt.modal
	rt.consumed = true
	rt.requestFrame()
	return true
}
