package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"slices"
)

type physicalShortcut struct {
	mods                   Modifiers
	code, second, identity string
	terminal               bool
	region                 string
}
type physicalShortcutState struct {
	regs, next, delivered      []physicalShortcut
	held                       map[string]bool
	consumed                   map[string]bool
	focus                      uint64
	mods                       Modifiers
	code                       string
	blocked                    bool
	dead                       bool
	terminals, nextTerminals   map[uint64]bool
	modalNames, nextModalNames map[uint64]string
}

// PhysicalShortcut observes one or two simultaneously held physical keys.
// It shares PageShortcut's native focus/overlay ownership and never installs
// a system hook. Unknown scan codes cannot be guessed from typed characters.
func (c *Context) PhysicalShortcut(mods Modifiers, code, second string, identity ...string) bool {
	raw := c.build()
	if raw == nil || raw.inert || code == "" {
		return false
	}
	rt := raw.rt
	rt.applyInputs()
	if second != "" && code > second {
		code, second = second, code
	}
	owner := ""
	if len(identity) > 0 {
		owner = identity[0]
	}
	terminal := len(identity) > 1 && identity[1] == "terminal"
	region := ""
	if len(identity) > 1 && (identity[1] == "application-fixed" || identity[1] == "page-close" || identity[1] == "page-resource" || identity[1] == "native-priority" || identity[1] == "native-no-modal" || identity[1] == "settings-open" || identity[1] == "shortcuts-open") {
		region = identity[1]
	}
	r := physicalShortcut{mods: mods, code: code, second: second, identity: owner, terminal: terminal, region: region}
	rt.physical.next = append(rt.physical.next, r)
	for i, d := range rt.physical.delivered {
		if d == r {
			rt.physical.delivered = append(rt.physical.delivered[:i], rt.physical.delivered[i+1:]...)
			if !rt.physicalRegistrationAllowed(r) {
				return false
			}
			rt.consumed = true
			return true
		}
	}
	return false
}

// Fixed application controls and close gestures have explicit owners even in
// editable/modal surfaces. They retain physical-key IME/dead/repeat rejection;
// ordinary PageShortcut and terminal registrations keep their original guard.
func (rt *engine) physicalRegistrationAllowed(r physicalShortcut) bool {
	// Native-priority may bypass ordinary editor ownership, never its IME.
	if focused := rt.states[rt.focused]; focused != nil {
		if focused.editor != nil && focused.editor.compose != "" {
			return false
		}
		if focused.textClient != nil {
			if _, marked := focused.textClient.MarkedRange(); marked {
				return false
			}
		}
	}
	if r.region == "native-priority" || r.region == "native-no-modal" || r.region == "settings-open" || r.region == "shortcuts-open" {
		if !rt.windowFocused {
			return false
		}
		for _, scope := range rt.focusScopes {
			if scope.anchor != 0 {
				return false
			}
		}
		if rt.modal == 0 {
			return true
		}
		name := rt.physical.modalNames[rt.modal]
		switch r.region {
		case "native-priority":
			// Native priority is an editable/terminal input policy, not
			// permission to execute behind an unrelated shell decision.
			return false
		case "settings-open":
			return name == "settings"
		case "shortcuts-open":
			return name == "settings" || name == "shortcuts"
		default:
			return false
		}
	}
	if r.region == "application-fixed" || r.region == "page-close" || r.region == "page-resource" {
		if !rt.windowFocused {
			return false
		}
		if r.region == "page-resource" {
			if rt.modal != 0 {
				return false
			}
			for _, scope := range rt.focusScopes {
				if scope.anchor != 0 {
					return false
				}
			}
		}
		if r.region == "application-fixed" {
			if h := rt.focusHandler(); h != nil && rt.physical.terminals[h.id] {
				return false
			}
		}
		return true
	}
	return rt.physicalCommandAllowed(r.terminal)
}

func (rt *engine) physicalEventAllowed() bool {
	if rt.physicalCommandAllowed(true) {
		return true
	}
	for _, registration := range rt.physical.regs {
		if registration.region != "" && rt.physicalRegistrationAllowed(registration) {
			return true
		}
	}
	return false
}

// TerminalShortcutScope marks the actual terminal input widget. It grants no
// command by itself; only registrations explicitly declaring terminal scope
// can claim keys from it. Other custom inputs and IME retain their ownership.
func TerminalShortcutScope(element Element) {
	node := element.node()
	if node == nil {
		return
	}
	p := &node.c.rt.physical
	if p.nextTerminals == nil {
		p.nextTerminals = map[uint64]bool{}
	}
	p.nextTerminals[node.id] = true
}
func (rt *engine) physicalCommandAllowed(terminal bool) bool {
	if rt.pageCommandAllowed() {
		return true
	}
	if !terminal || !rt.windowFocused || rt.modal != 0 {
		return false
	}
	for _, scope := range rt.focusScopes {
		if scope.anchor != 0 {
			return false
		}
	}
	h := rt.focusHandler()
	return h != nil && rt.physical.terminals[h.id]
}

// PageCommandsAllowed is the same live native overlay/focus guard used by
// keyboard dispatch, exposed for revision-checked explicit menu gestures.
func (c *Context) PageCommandsAllowed() bool {
	raw := c.build()
	return raw != nil && !raw.inert && raw.rt.pageCommandAllowed()
}
func (c *Context) MenuCommandsAllowed() bool {
	raw := c.build()
	return raw != nil && raw.rt.windowFocused && c.MenuOverlayClear()
}

// MenuOverlayClear is for an explicitly captured owned child-window focus;
// its caller must independently verify that exact native child's live focus.
func (c *Context) MenuOverlayClear() bool {
	raw := c.build()
	if raw == nil || raw.inert || raw.rt.modal != 0 {
		return false
	}
	for _, scope := range raw.rt.focusScopes {
		if scope.anchor != 0 {
			return false
		}
	}
	// Deferred application decisions run before commitElement updates rt.modal.
	// Inspect this build too, including a dialog opened by this frame's input.
	var blocked func(*node) bool
	blocked = func(element *node) bool {
		if element == nil || element.flags&(flagInvisible|flagInert) != 0 {
			return false
		}
		if element.flags&flagModal != 0 || element.popover != nil {
			return true
		}
		for child := element.first; child != nil; child = child.next {
			if blocked(child) {
				return true
			}
		}
		return false
	}
	if blocked(raw.root) || blocked(raw.overlay) {
		return false
	}
	return true
}
func (rt *engine) resetPhysicalKeys() {
	rt.physical.held = nil
	rt.physical.consumed = nil
	rt.physical.delivered = nil
	rt.physical.dead = false
}
func (rt *engine) acceptPhysicalShortcuts() {
	if !slices.Equal(rt.physical.regs, rt.physical.next) {
		rt.resetPhysicalKeys()
	}
	rt.physical.regs, rt.physical.next = rt.physical.next, rt.physical.regs
	rt.physical.terminals = rt.physical.nextTerminals
	rt.physical.nextTerminals = nil
	rt.physical.modalNames = rt.physical.nextModalNames
	rt.physical.nextModalNames = nil
}
func (rt *engine) physicalKeyEvent(ev platform.SurfaceEvent) bool {
	p := &rt.physical
	p.code, p.blocked = ev.Code, ev.ShortcutBlocked
	if ev.ShortcutDead {
		p.dead = true
		p.blocked = true
	} else if p.dead && ev.Kind == platform.KeyPressed {
		p.dead = false
		p.blocked = true
	}
	mods := Modifiers(ev.Mods)
	if p.blocked || ev.Code == "" || !rt.physicalEventAllowed() {
		p.held = nil
		p.consumed = nil
		return false
	}
	if p.focus != rt.focused || p.mods != mods {
		p.held = nil
		p.focus, p.mods = rt.focused, mods
	}
	if ev.Kind == platform.KeyReleased {
		consume := p.consumed[ev.Code]
		delete(p.consumed, ev.Code)
		delete(p.held, ev.Code)
		return consume
	}
	if ev.Kind != platform.KeyPressed {
		return false
	}
	if ev.Repeat {
		_, consumed := p.consumed[ev.Code]
		return consumed
	}
	delete(p.consumed, ev.Code)
	if p.held == nil {
		p.held = map[string]bool{}
	}
	p.held[ev.Code] = true
	for _, r := range p.regs {
		if r.mods != mods || !rt.physicalRegistrationAllowed(r) {
			continue
		}
		match := r.second == "" && r.code == ev.Code || r.second != "" && len(p.held) == 2 && p.held[r.code] && p.held[r.second]
		if !match {
			continue
		}
		if p.consumed == nil {
			p.consumed = map[string]bool{}
		}
		if r.second != "" {
			p.consumed[r.code], p.consumed[r.second] = false, false
		}
		p.consumed[ev.Code] = true
		p.held = nil
		p.delivered = append(p.delivered, r)
		rt.requestFrame()
		return true
	}
	return false
}

// ShortcutModalScope declares the top-level product modal's command owner.
// A nested unmarked confirmation remains other and cannot inherit permission.
func ShortcutModalScope(element Element, name string) {
	n := element.node()
	if n == nil {
		return
	}
	p := &n.c.rt.physical
	for parent := n; parent != nil; parent = parent.parent {
		if parent.flags&flagModal != 0 {
			n = parent
			break
		}
	}
	if p.nextModalNames == nil {
		p.nextModalNames = map[uint64]string{}
	}
	p.nextModalNames[n.id] = name
}

// ShortcutModalName reports the actual top modal, not a page-level open flag.
// Unmarked nested confirmations remain other even over settings.
func (c *Context) ShortcutModalName() string {
	raw := c.build()
	if raw == nil || raw.inert || !raw.rt.windowFocused {
		return ""
	}
	for _, scope := range raw.rt.focusScopes {
		if scope.anchor != 0 {
			return "other"
		}
	}
	// A command deferred to this build must see a new nested decision before
	// commitElement replaces rt.modal. Never inherit the outer modal's name.
	var modal *node
	popover := false
	var visit func(*node)
	visit = func(element *node) {
		if element == nil || element.flags&(flagInvisible|flagInert) != 0 {
			return
		}
		if element.flags&flagModal != 0 {
			modal = element
		}
		popover = popover || element.popover != nil
		for child := element.first; child != nil; child = child.next {
			visit(child)
		}
	}
	visit(raw.root)
	visit(raw.overlay)
	if popover {
		return "other"
	}
	if modal != nil {
		if name := raw.rt.physical.nextModalNames[modal.id]; name != "" {
			return name
		}
		return "other"
	}
	if raw.rt.modal == 0 {
		return ""
	}
	if name := raw.rt.physical.modalNames[raw.rt.modal]; name != "" {
		return name
	}
	return "other"
}
