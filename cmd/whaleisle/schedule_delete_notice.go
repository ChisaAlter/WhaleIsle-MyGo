package main

import (
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

const scheduleDeleteToastID = "schedule-delete-outcome"

// The app owns this seat so closing a task or leaving its Session cannot hide
// the outcome. It consumes only an already-settled Store result; no RPC is sent.
func (s *mainState) syncScheduleDeleteNotice(c *ui.Context) {
	s.mu.Lock()
	store := s.schedule
	s.mu.Unlock()
	if store == nil {
		c.CloseToast(scheduleDeleteToastID)
		return
	}
	notice, fresh := store.ConsumeDeletionNotice()
	if notice.Sequence == 0 {
		c.CloseToast(scheduleDeleteToastID)
		return
	}
	if !fresh {
		return
	}
	text, tone := i18n.T(i18n.KeyCmdTaskDeleted), "success"
	if notice.Outcome == "failed" {
		text, tone = i18n.T(i18n.KeyCmdCouldNotDeleteTheTask), "warning"
	}
	c.AddToast(ui.Toast{ID: scheduleDeleteToastID, Description: text, Type: tone, Timeout: 4 * time.Second,
		OnClose: func() { store.DismissDeletionNotice(notice) },
	})
}
