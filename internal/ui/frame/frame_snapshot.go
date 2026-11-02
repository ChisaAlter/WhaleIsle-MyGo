package frame

import "whaleisle/internal/store/session"

// renderSnapshot is only for synchronous view construction. The supplied pass
// belongs to the outer render owner and must never be retained by an action or
// asynchronous reader. Every call still checks the Store's current revision.
func renderSnapshot(m Model) session.ViewState {
	if m.FrameView.BelongsTo(m.Store) {
		return m.FrameView.Snapshot()
	}
	return m.Store.Snapshot()
}
