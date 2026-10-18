package mygo

import "strconv"

// ProductKeyBinding is an accepted physical-key registry binding from one
// application-owned product document. Two distinct codes are held, not sequenced.
type ProductKeyBinding struct {
	Code       string
	Mods       uint8
	SecondCode string
}
type productKeyboardState struct {
	revision string
	active   bool
	bindings []ProductKeyBinding
	dispatch func(string, string, uint8, string)
	held     map[uint32]bool
	consumed map[uint32]bool // value marks whether its initial press was intercepted
	mods     uint8
}

// SetProductKeyboard replaces the exact product page's accepted snapshot.
// Authenticate the product origin and revoke on navigation; never opt ordinary
// websites or isolated document previews into this capability.
func (p *Page) SetProductKeyboard(revision string, active bool, keys []ProductKeyBinding, dispatch func(string, string, uint8, string)) {
	keys = append([]ProductKeyBinding(nil), keys...)
	onMain(func() {
		accepted := []ProductKeyBinding{}
		for _, k := range keys {
			if k.Mods <= 15 && productCodeVK(k.Code) != 0 && (k.SecondCode == "" || (k.SecondCode != k.Code && productCodeVK(k.SecondCode) != 0)) {
				accepted = append(accepted, k)
			}
		}
		p.w.productKeyboard = &productKeyboardState{revision: revision, active: active, bindings: accepted, dispatch: dispatch}
		if native, ok := p.w.native.(interface{ ResetProductKeyboard() }); ok {
			native.ResetProductKeyboard()
		}
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
	if vk := map[string]uint32{"Escape": 27, "Space": 32, "Tab": 9, "Backspace": 8, "Delete": 46, "ArrowLeft": 37, "ArrowUp": 38, "ArrowRight": 39, "ArrowDown": 40}[code]; vk != 0 {
		return vk
	}
	if len(code) > 1 && code[0] == 'F' {
		if n, err := strconv.Atoi(code[1:]); err == nil && n >= 1 && n <= 24 {
			return uint32(111 + n)
		}
	}
	return webShortcutVirtualKey(code)
}

// ProductWebKey is the down-only compatibility entry point for native backends.
func (h *windowHandler) ProductWebKey(vk uint32, ctrl, alt, shift, super, repeat bool) bool {
	return h.ProductWebKeyEvent(vk, ctrl, alt, shift, super, repeat, true)
}
func (h *windowHandler) ProductWebKeyReset() {
	if s := h.w.productKeyboard; s != nil {
		s.held = nil
		s.consumed = nil
	}
}

// ProductWebKeyEvent accepts only real native input. Releases, modifier changes,
// ownership changes and accepted snapshots cannot seed a later chord.
func (h *windowHandler) ProductWebKeyEvent(vk uint32, ctrl, alt, shift, super, repeat, down bool) bool {
	s := h.w.productKeyboard
	if s == nil || !s.active || s.revision == "" || s.dispatch == nil {
		h.ProductWebKeyReset()
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
	if super {
		mods |= 8
	}
	if s.held == nil {
		s.held = map[uint32]bool{}
	}
	if s.consumed == nil {
		s.consumed = map[uint32]bool{}
	}
	if s.mods != mods {
		s.held = map[uint32]bool{}
		s.mods = mods
	}
	modifier := vk == 16 || vk == 17 || vk == 18 || vk == 91 || vk == 92 || (vk >= 160 && vk <= 165)
	if !down {
		consume := s.consumed[vk]
		delete(s.consumed, vk)
		delete(s.held, vk)
		if modifier {
			s.held = map[uint32]bool{}
		}
		return consume
	}
	if modifier {
		s.held = map[uint32]bool{}
		return false
	}
	if repeat {
		if _, ok := s.consumed[vk]; ok {
			return true
		}
	} else {
		delete(s.consumed, vk)
	}
	if !repeat {
		s.held[vk] = true
	}
	for _, k := range s.bindings {
		first, second := productCodeVK(k.Code), productCodeVK(k.SecondCode)
		match := k.Mods == mods && ((second == 0 && first == vk) || (second != 0 && !repeat && len(s.held) == 2 && s.held[first] && s.held[second] && (vk == first || vk == second)))
		if !match {
			continue
		}
		if !repeat {
			if second != 0 {
				s.consumed[first] = false
				s.consumed[second] = false
			}
			s.consumed[vk] = true
			s.held = map[uint32]bool{}
			s.dispatch(s.revision, k.Code, mods, k.SecondCode)
		}
		return true
	}
	return false
}
