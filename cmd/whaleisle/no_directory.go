package main

import (
	"context"
	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
)

// Keep the current draft on failure and never steal focus from newer navigation.
func (s *mainState) openNoDirectory(win *mygo.Window, sessions *session.Store, draft *composer.DraftState, sourceID string) {
	if win == nil || sessions == nil || draft.Sending || draft.Preparing > 0 {
		return
	}
	carry := sourceID == ""
	snap := sessions.Snapshot()
	for _, it := range snap.Items {
		if it.SessionID == sourceID {
			carry = it.Blank && len(snap.Messages) == 0 && draft.EditSeq == 0
			break
		}
	}
	s.mu.Lock()
	var excluded []string
	for id, d := range s.composerDrafts {
		if d.Text != "" || len(d.Attachments) > 0 || d.Sending || d.Preparing > 0 || d.EditSeq > 0 {
			excluded = append(excluded, id)
		}
	}
	s.mu.Unlock()
	draft.Sending = true
	draft.Error = ""
	go func() {
		id, err := sessions.CreateNoDirectorySession(context.Background(), excluded...)
		win.Update(func() {
			draft.Sending = false
			if err != nil {
				draft.Error = err.Error()
				// Only a failure leaves a status, in the shell's locale: the old
				// success path painted "no workspace folder ready" over the hero.
				sessions.SetLastAction(i18n.T(i18n.KeyComposerDirNoWorkspace) + ": " + err.Error())
				return
			}
			if sessions.SelectedSessionID() != sourceID {
				return
			}
			if carry && id != sourceID {
				s.mu.Lock()
				target := s.composerDrafts[id]
				if target == nil || (target.Text == "" && len(target.Attachments) == 0 && !target.Sending && target.Preparing == 0 && target.EditSeq == 0) {
					s.composerDrafts[id] = draft
					delete(s.composerDrafts, sourceID)
				}
				s.mu.Unlock()
			}
			sessions.Select(id)
			// No status on success: the conversation itself is the feedback.
			sessions.SetLastAction("")
		})
	}()
}
