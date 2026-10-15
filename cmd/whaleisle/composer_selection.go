package main

import (
	"errors"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// appendSelectionToComposer uses the exact session/draft captured by this view.
// Navigation or an in-flight send cannot redirect the selection to another chat.
func appendSelectionToComposer(sessions *session.Store, id string, draft *composer.DraftState, fragment string) error {
	if sessions == nil || id == "" || sessions.Snapshot().SelectedID != id {
		return errors.New("select a conversation before adding text to its draft")
	}
	return composer.AppendSelectionDraft(draft, fragment)
}
