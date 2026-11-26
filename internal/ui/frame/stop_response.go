package frame

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func cancelCapturedResponseStore(store *session.Store, snap session.ViewState) {
	if store == nil {
		return
	}
	live := store.Published().Value
	// Bind both keyboard regions and the button to their captured session. A
	// replacement turn, follow or Host must not inherit the previous gesture.
	if live.SelectedID != snap.SelectedID || live.ComposerHostGeneration != snap.ComposerHostGeneration || live.StopGeneration != snap.StopGeneration || live.StopTurnSeq != snap.StopTurnSeq {
		return
	}
	store.CancelSession(context.Background(), snap.SelectedID)
}

func cancelCapturedResponse(m Model, snap session.ViewState) {
	cancelCapturedResponseStore(m.Store, snap)
}

func bindChatStopRegion(area ui.Element, m *Model, snap session.ViewState) {
	client := m.ComposerTextInput
	if client == nil && m.ComposerState != nil {
		client = m.ComposerState.Client
	}
	if client == nil {
		return
	}
	store := m.Store
	interactions := m.Interactions
	selectedID := snap.SelectedID
	composer.BindStopRegion(area, client, composer.Model{Snapshot: snap, StopBlocked: hasPendingStopInteraction(interactions, selectedID), OnStop: func() {
		if !hasPendingStopInteraction(interactions, selectedID) {
			cancelCapturedResponseStore(store, snap)
		}
	}}, true)
}
