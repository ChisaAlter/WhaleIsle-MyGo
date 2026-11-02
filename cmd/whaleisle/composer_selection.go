package main

import (
	"errors"
	"whaleisle/internal/ui/i18n"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// appendSelectionToComposer uses the exact session/draft captured by this view.
// Navigation or an in-flight send cannot redirect the selection to another chat.
func appendSelectionToComposer(sessions *session.Store, id string, draft *composer.DraftState, fragment string) error {
	if sessions == nil || id == "" || sessions.SelectedSessionID() != id {
		return errors.New(i18n.T(i18n.KeyCmdComposerSelectConversation))
	}
	return composer.AppendSelectionDraft(draft, fragment)
}
