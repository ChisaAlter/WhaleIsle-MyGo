package mygo

// ProductKeyBinding is an accepted, single-key registry binding from one
// application-owned product document. It is not a command or an RPC name.
type ProductKeyBinding struct {
	Code string
	Mods uint8
}
type productKeyboardState struct {
	revision string
	active   bool
	bindings map[webShortcutKey]string
	dispatch func(string, string, uint8)
}

// SetProductKeyboard updates only the exact product page owning this snapshot.
// The caller must authenticate the product origin and revoke on navigation.
// Ordinary websites and local document previews must never opt in.
func (p *Page) SetProductKeyboard(revision string, active bool, keys []ProductKeyBinding, dispatch func(string, string, uint8)) {
	keys = append([]ProductKeyBinding(nil), keys...)
	onMain(func() {
		bindings := map[webShortcutKey]string{}
		for _, key := range keys {
			if vk := productCodeVK(key.Code); vk != 0 && key.Mods > 0 && key.Mods <= 7 {
				bindings[webShortcutKey{vk, key.Mods}] = key.Code
			}
		}
		p.w.productKeyboard = &productKeyboardState{revision, active, bindings, dispatch}
	})
}
func productCodeVK(code string) uint32 {
	if len(code) == 4 && code[:3] == "Key" {
		return webShortcutVirtualKey(code[3:])
	}
	if len(code) == 6 && code[:5] == "Digit" {
		return webShortcutVirtualKey(code[5:])
	}
	names := map[string]string{"Comma": ",", "Period": ".", "Slash": "/", "Backslash": "\\", "Backquote": "`", "Semicolon": ";", "Quote": "'", "BracketLeft": "[", "BracketRight": "]", "Minus": "-", "Equal": "="}
	if key := names[code]; key != "" {
		return webShortcutVirtualKey(key)
	}
	return webShortcutVirtualKey(code)
}

// Only the WebView2 controller's real input callback calls this method. Returning
// true consumes the native browser key before the renderer's default action.
func (h *windowHandler) ProductWebKey(vk uint32, ctrl, alt, shift, super, repeat bool) bool {
	s := h.w.productKeyboard
	if s == nil || !s.active || s.revision == "" || s.dispatch == nil || super {
		return false
	}
	mods := uint8(0)
	if ctrl {
		mods |= 1
	}
	if alt {
		mods |= 2
	}
	if shift {
		mods |= 4
	}
	code := s.bindings[webShortcutKey{vk, mods}]
	if code == "" {
		return false
	}
	if !repeat {
		s.dispatch(s.revision, code, mods)
	}
	return true
}
