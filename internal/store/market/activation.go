package market

// Activation remembers changes already written to the profile which need a
// fresh Host generation. A restart request or an old Ready callback is not
// evidence that the saved changes have loaded.
type Activation struct {
	Pending, Restarting, Failed, Refreshed bool
	generation                             int
}

func (a *Activation) Require(generation int) {
	*a = Activation{Pending: true, generation: generation}
}

// Observe completes only on a genuinely newer, ready Host. Failed restarts
// retain the pending change so the operator can retry without reinstalling.
func (a *Activation) Observe(generation int, state string) bool {
	if !a.Pending || generation < a.generation {
		return false
	}
	if state == "ready" {
		if generation <= a.generation {
			return false
		}
		a.Pending, a.Restarting, a.Failed, a.Refreshed = false, false, false, true
		return true
	}
	a.Restarting = state == "starting"
	a.Failed = state == "failed"
	return false
}
