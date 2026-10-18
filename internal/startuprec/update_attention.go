package startuprec

// UpdateAttentionScope binds a reminder and a queued confirmation to one live
// window, Host generation and pinned native catalog. It carries no authority to
// download, install or stop tasks.
type UpdateAttentionScope struct {
	Window, Host        string
	HostGeneration      int
	CatalogGeneration   uint64
	Route, Tag, Version string
}

type UpdateAttentionTicket struct {
	Scope UpdateAttentionScope
	epoch uint64
}

// UpdateAttention is used under the owner's state lock. Native notification
// failures cannot consume Pending: the visible launcher remains the fallback.
type UpdateAttention struct {
	ticket             UpdateAttentionTicket
	pending, prompting bool
	epoch              uint64
	reminded, seen     map[string]bool
}

func (s UpdateAttentionScope) valid() bool { return s.Window != "" && s.Tag != "" && s.Version != "" }
func (s UpdateAttentionScope) key() string { return s.Route + "\x00" + s.Version }

// Observe returns one best-effort reminder per route/version, even across focus,
// catalog refresh and Host/window replacement. A disabled preference is an
// explicit ignore of that version. Invalidation alone does not consume the ask.
func (a *UpdateAttention) Observe(scope UpdateAttentionScope, enabled, focused bool) (UpdateAttentionTicket, bool) {
	if !scope.valid() {
		a.Clear()
		return UpdateAttentionTicket{}, false
	}
	if a.seen == nil {
		a.seen = map[string]bool{}
	}
	if !enabled {
		a.seen[scope.key()] = true
	}
	if a.seen[scope.key()] {
		a.Clear()
		return UpdateAttentionTicket{}, false
	}
	if !a.pending || a.ticket.Scope != scope {
		a.epoch++
		a.ticket = UpdateAttentionTicket{Scope: scope, epoch: a.epoch}
		a.pending, a.prompting = true, false
	}
	if a.reminded == nil {
		a.reminded = map[string]bool{}
	}
	remind := !a.reminded[scope.key()] && !focused
	a.reminded[scope.key()] = true
	return a.ticket, remind
}

func (a *UpdateAttention) Current(t UpdateAttentionTicket) bool {
	return a.pending && t.epoch != 0 && t == a.ticket
}
func (a *UpdateAttention) Pending() bool   { return a.pending }
func (a *UpdateAttention) Prompting() bool { return a.prompting }
func (a *UpdateAttention) BeginPrompt(t UpdateAttentionTicket) bool {
	if !a.Current(t) || a.prompting {
		return false
	}
	a.prompting = true
	return true
}
func (a *UpdateAttention) CancelPrompt(t UpdateAttentionTicket) {
	if a.Current(t) {
		a.prompting = false
	}
}

// Resolve consumes both acceptance and Later/Cancel. The caller separately
// checks the explicit dialog response and live scope before downloading.
func (a *UpdateAttention) Resolve(t UpdateAttentionTicket) bool {
	if !a.Current(t) || !a.prompting {
		return false
	}
	a.DismissCurrent()
	return true
}
func (a *UpdateAttention) DismissCurrent() {
	if a.pending {
		if a.seen == nil {
			a.seen = map[string]bool{}
		}
		a.seen[a.ticket.Scope.key()] = true
	}
	a.Clear()
}
func (a *UpdateAttention) Clear() {
	a.ticket = UpdateAttentionTicket{}
	a.pending, a.prompting = false, false
}
