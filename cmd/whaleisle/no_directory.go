package main

import (
	"context"
	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// Keep the current draft on failure and never steal focus from newer navigation.
func (s *mainState) openNoDirectory(win *mygo.Window, sessions *session.Store, draft *composer.DraftState, sourceID string) {
	if win == nil || sessions == nil || draft == nil || draft.Sending || draft.Preparing > 0 {
		return
	}
	s.revealMainConversation()
	carry := sourceID == ""
	snap := sessions.Snapshot()
	if snap.SelectedID != sourceID {
		return
	}
	for _, it := range snap.Items {
		if it.SessionID == sourceID {
			carry = it.Blank && !snap.HasConversationContent() && draft.EditSeq == 0
			break
		}
	}
	s.mu.Lock()
	owned := s.composerDrafts[sourceID] == draft
	s.mu.Unlock()
	if !owned {
		return
	}
	draft.Error = ""
	op, err := s.captureWorkspaceSessionOpen(win, sessions, snap.WorkspaceLayout.HostGeneration, "::scratch")
	if err != nil || op == nil {
		return
	}
	op.carryDraft = carry
	go func() {
		id, err := op.target.Create(context.Background())
		if win.IsDestroyed() {
			return
		}
		win.Update(func() {
			if err != nil && op.current() {
				draft.Error = err.Error()
			}
			if op.finish(id, err) {
				// No status on success: the conversation is the feedback.
				sessions.SetLastAction("")
			}
		})
	}()
}
