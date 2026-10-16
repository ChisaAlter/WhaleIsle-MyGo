package ui

// ObserveUnhandledEscape installs a region-local fixed Escape sequence hook.
// eligible is true only for an unclaimed, non-repeated plain Escape after the
// focused widget's input handler/editor. Other keys and consumed Escape notify
// with eligible=false so a pending first press cannot survive local dismissal.
// Only the nearest registered ancestor of focus may consume; overlays, IME and
// foreign custom input widgets never yield Escape to this hook.
func (e Element) ObserveUnhandledEscape(fn func(eligible bool) bool) Element {
	if n := e.node(); n != nil {
		n.regionEscape = fn
	}
	return e
}

func (rt *engine) regionEscapeInput(eligible bool) bool {
	var owner *state
	if eligible && rt.windowFocused && rt.modal == 0 {
		blocked := false
		for _, scope := range rt.focusScopes {
			if scope.anchor != 0 {
				blocked = true
				break
			}
		}
		focus := rt.states[rt.focused]
		if focus != nil && (focus.input != nil || focus.textClient != nil || focus.editor != nil && focus.editor.compose != "") {
			blocked = true
		}
		if !blocked {
			for s := focus; s != nil; s = rt.states[s.parent] {
				if s.flags&(flagDisabled|flagInert) != 0 {
					break
				}
				if s.regionEscape != nil {
					owner = s
					break
				}
				if s.parent == 0 {
					break
				}
			}
		}
	}
	// Reset non-owning regions too, including a previous conversation occurrence.
	for _, s := range rt.states {
		if s.regionEscape != nil && s != owner {
			s.regionEscape(false)
		}
	}
	return owner != nil && owner.regionEscape(true)
}

// EscapeSequenceAvailable lets a custom text client defer to a dialog/popover
// before handling its own fixed sequence. It does not consume any input.
func (c *Context) EscapeSequenceAvailable() bool {
	rt := c.runtime()
	if rt == nil || !rt.windowFocused || rt.modal != 0 {
		return false
	}
	for _, scope := range rt.focusScopes {
		if scope.anchor != 0 {
			return false
		}
	}
	return true
}
