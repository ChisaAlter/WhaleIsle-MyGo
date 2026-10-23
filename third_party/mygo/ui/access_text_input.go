package ui

import (
	"slices"
	"unicode/utf8"

	"github.com/egoist/mygo/internal/platform"
)

// TextInputAccessibility describes an application-owned text field to assistive
// technology. These properties describe the client's editing policy; the client
// remains responsible for enforcing it for keyboard and native input too.
type TextInputAccessibility struct {
	Placeholder string
	Multiline   bool
	ReadOnly    bool
	Password    bool
}

// AccessibleTextInputClient opts an application-owned TextInputClient into
// accessible text queries and full-value replacement. Pair HandleTextInput with
// RoleTextField (or RoleComboBox) and a Label. Clients without this interface keep
// their explicit element Value and do not offer an accessible editing action.
//
// TextForRange supplies the entire document, including marked text, for the range
// [0, max int). ReplaceText clamps that same range to the document when assistive
// technology sets its value. Replacement uses the client's own undo policy. A
// password client's text is never queried for the accessibility tree.
type AccessibleTextInputClient interface {
	TextInputClient
	TextInputAccessibility() TextInputAccessibility
}

func accessTextInput(e *node, n *platform.AccessNode, disabled bool) {
	if n.Role != platform.RoleTextField && n.Role != platform.RoleComboBox {
		return
	}
	client, ok := e.textClient.(AccessibleTextInputClient)
	if !ok {
		return
	}
	info := client.TextInputAccessibility()
	n.Placeholder = info.Placeholder
	if info.Multiline {
		n.States |= platform.AccessMultiline
	}
	if info.ReadOnly {
		n.States |= platform.AccessReadOnly
	} else if !disabled {
		n.Actions |= platform.ActionSetValue
	}
	if info.Password {
		n.States |= platform.AccessPassword
		n.Value, n.SelStart, n.SelEnd = "", 0, 0
		return
	}
	n.Value, _ = client.TextForRange(TextInputRange{End: int(^uint(0) >> 1)})
	sel := platform.NormalizeTextRange(client.Selection().Range)
	// AccessNode selections are rune offsets. Native input clients use UTF-16;
	// the platform accessibility backends perform their own native conversion.
	n.SelStart = utf8.RuneCountInString(n.Value[:UTF16ByteOffset(n.Value, sel.Start)])
	n.SelEnd = utf8.RuneCountInString(n.Value[:UTF16ByteOffset(n.Value, sel.End)])
}

func (rt *engine) accessSetTextInputValue(s *state, value string) {
	client, ok := s.textClient.(AccessibleTextInputClient)
	if !ok || rt.closed || rt.textInputClosed || s.seen != rt.frame || s.pass != rt.pass || client.TextInputAccessibility().ReadOnly {
		return
	}
	id, adapter := s.id, s.textAdapter
	// Retained router pages and controls behind a modal are not editable through
	// a cached accessibility reference. The current tree owns the admission.
	current := func() bool {
		if rt.closed || rt.textInputClosed || rt.states[id] != s || s.id != id || s.textAdapter != adapter || s.textClient != client || s.seen != rt.frame || s.pass != rt.pass || client.TextInputAccessibility().ReadOnly {
			return false
		}
		return slices.ContainsFunc(rt.accessTree().Nodes, func(n platform.AccessNode) bool {
			return n.ID == id && n.Actions&platform.ActionSetValue != 0
		})
	}
	if !current() {
		return
	}
	defer func() {
		if !rt.closed && !rt.textInputClosed {
			rt.requestFrame()
			rt.updateTextInput()
		}
	}()
	if rt.ime.state.Client != nil {
		if old, ok := rt.ime.state.Client.(*textInputAdapter); ok && old.id != s.id {
			old.release()
		}
		// A full-value replacement invalidates pending preedit/commit offsets,
		// even when this is the same focused client. Temporarily remove focus so
		// native cancellation cannot reenter its previous text-input adapter.
		rt.focused = 0
		rt.ime.state = platform.TextInputState{}
		rt.host.setTextInput(rt.ime.state)
	}
	// Native reset can reenter the host. Do not call an adapter removed/replaced
	// by that callback, or a window that has just closed.
	if !current() {
		return
	}
	rt.focusOn(s)
	r := TextInputRange{End: int(^uint(0) >> 1)}
	client.ReplaceText(&r, value)
}
