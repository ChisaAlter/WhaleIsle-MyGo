package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func cancelCapturedResponse(m Model, snap session.ViewState) {
	live := m.Store.Snapshot()
	// Bind both keyboard regions and the button to their captured session. A
	// replacement turn, follow or Host must not inherit the previous gesture.
	if live.SelectedID != snap.SelectedID || live.ComposerHostGeneration != snap.ComposerHostGeneration || live.StopGeneration != snap.StopGeneration || live.StopTurnSeq != snap.StopTurnSeq {
		return
	}
	m.Store.CancelSession(context.Background(), snap.SelectedID)
}

func bindChatStopRegion(area ui.Element, m Model, snap session.ViewState) {
	client := m.ComposerTextInput
	if client == nil && m.ComposerState != nil {
		client = m.ComposerState.Client
	}
	composer.BindStopRegion(area, client, composer.Model{Snapshot: snap, StopBlocked: hasPendingStopInteraction(m.Interactions, snap.SelectedID), OnStop: func() {
		if !hasPendingStopInteraction(m.Interactions, snap.SelectedID) {
			cancelCapturedResponse(m, snap)
		}
	}}, true)
}
