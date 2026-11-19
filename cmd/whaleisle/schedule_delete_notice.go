package main

import (
	"whaleisle/internal/ui/i18n"
)

// syncScheduleDeleteNotice reports a failed deletion where the person can see it
// from any page: the conversation's status line, which outlives the Session the
// task was deleted from. A confirmed deletion is already visible when the
// authoritative catalog drops its row, so it needs no line of its own.
//
// It consumes only an already-settled Store result; no RPC is sent.
func (s *mainState) syncScheduleDeleteNotice() {
	s.mu.Lock()
	store, sessions := s.schedule, s.sessions
	s.mu.Unlock()
	if store == nil || sessions == nil {
		return
	}
	notice, fresh := store.ConsumeDeletionNotice()
	if notice.Sequence == 0 || !fresh {
		return
	}
	if notice.Outcome != "failed" {
		store.DismissDeletionNotice(notice)
		return
	}
	sessions.SetLastAction(i18n.T(i18n.KeyCmdCouldNotDeleteTheTask))
	store.DismissDeletionNotice(notice)
}
