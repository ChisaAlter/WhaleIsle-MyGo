package interaction

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
)

// Only the explicitly focused approval body and its buttons install this
// handler. Editors, terminal/web guests and modal focus keep their own keys.
// The state and node identity belong to the request, never its list position.
type approvalKeys struct{ composing, compositionEnded bool }

func approvalHandler(keys *approvalKeys, pending interaction.Pending, store *interaction.Store, button bool) func(ui.InputEvent) bool {
	return func(ev ui.InputEvent) bool {
		return keys.handle(ev, pending, button, func() interaction.ViewState { return store.SnapshotForAgent(pending.AgentID) }, store.ResolveAllow, store.ResolveDeny)
	}
}

func (s *approvalKeys) handle(ev ui.InputEvent, pending interaction.Pending, button bool, snapshot func() interaction.ViewState, allow, deny func(string)) bool {
	switch ev.Kind {
	case ui.InputCompose:
		s.composing = ev.Text != ""
		s.compositionEnded = !s.composing
		return false
	case ui.InputText:
		if s.composing {
			s.composing, s.compositionEnded = false, true
		}
		return false
	case ui.InputKeyUp:
		s.compositionEnded = false
		return false
	}
	if ev.Kind != ui.InputKeyDown || ev.Mods != 0 || (ev.Key != ui.KeyEnter && ev.Key != ui.KeyEscape) {
		return false
	}
	// Enter on an action keeps the normal native button activation, including
	// Copy and Deny; it must never be reinterpreted as Allow.
	if button && ev.Key == ui.KeyEnter {
		return false
	}
	if ev.Repeat || s.composing || s.compositionEnded {
		return true
	}
	if pending.Event != interaction.EventApproval || pending.EventID == "" || pending.AgentID == "" {
		return true
	}
	// Revalidate at dispatch, not only render time. Error cards expose Retry
	// exclusively, preserving the previously selected outcome after failure.
	for _, current := range snapshot().Pending {
		if current.EventID != pending.EventID || current.AgentID != pending.AgentID || current.Event != interaction.EventApproval {
			continue
		}
		if current.Sending || current.Error != "" || current.Continued {
			return true
		}
		if ev.Key == ui.KeyEnter {
			allow(current.EventID)
		} else {
			deny(current.EventID)
		}
		return true
	}
	return true
}
