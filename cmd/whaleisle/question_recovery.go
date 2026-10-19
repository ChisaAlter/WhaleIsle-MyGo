package main

import (
	"encoding/json"
	"errors"
	"path/filepath"

	"whaleisle/internal/paths"
	"whaleisle/internal/recovery"
	"whaleisle/internal/store/interaction"
)

// syncDialogue only attaches the supervised local web profile. Its persistent
// identity survives restarts without storing the ephemeral endpoint or cookie.
func questionDraftProfileOwner() string {
	profile, err := filepath.Abs(paths.WebProfileDir())
	if err != nil {
		return ""
	}
	return "local-profile:" + filepath.ToSlash(filepath.Clean(profile))
}

func (s *mainState) bindQuestionDraftRecovery(inter *interaction.Store) {
	if inter == nil {
		return
	}
	s.mu.Lock()
	store := s.resumeStore
	s.mu.Unlock()
	owner := questionDraftProfileOwner()
	inter.SetQuestionDraftPersistence(owner, func(raw json.RawMessage) error {
		if store == nil {
			return errors.New("resume storage is not available")
		}
		return store.Update(func(state *recovery.State) {
			state.QuestionDrafts = append(json.RawMessage(nil), raw...)
		})
	})
	if store != nil {
		inter.RestoreQuestionDrafts(owner, store.Snapshot().QuestionDrafts)
	}
}
