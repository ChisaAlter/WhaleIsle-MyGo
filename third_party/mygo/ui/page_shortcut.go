package ui

// PageShortcut is a conservative native-page command. It never claims keys
// from editable/selectable text, custom input widgets (including terminals),
// dialogs or popovers. It is not a system-wide or WebView accelerator.
// The guard is checked at input dispatch as well as registration, so changing
// focus between frames does not steal the first key from an editor.
func (c *Context) PageShortcut(mods Modifiers, key Key) bool {
	raw := c.build()
	if raw == nil || raw.inert {
		return false
	}
	rt := raw.rt
	rt.applyInputs()
	rt.nextRegs = append(rt.nextRegs, shortcutReg{mods: mods, key: key, pageOnly: true})
	for i, d := range rt.delivered {
		if d.id == 0 && d.mods == mods && d.key == key {
			rt.delivered = append(rt.delivered[:i], rt.delivered[i+1:]...)
			if !rt.pageCommandAllowed() {
				return false
			}
			rt.consumed = true
			return true
		}
	}
	return false
}

func (rt *engine) pageCommandAllowed() bool {
	if !rt.windowFocused || rt.modal != 0 {
		return false
	}
	for _, sc := range rt.focusScopes {
		if sc.anchor != 0 {
			return false
		}
	}
	s := rt.states[rt.focused]
	return s == nil || (s.editor == nil && s.textClient == nil && s.input == nil && s.flags&(flagEditable|flagSelectable) == 0)
}
