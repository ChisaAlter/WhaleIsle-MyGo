package main

import (
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

const scheduleDeleteToastID = "schedule-delete-outcome"

type scheduleDeleteNoticeSeat struct{ showing bool }

// The app owns this seat so closing a task or leaving its Session cannot hide
// its failure. It consumes only an already-settled Store result; no RPC is sent.
func (s *mainState) syncScheduleDeleteNotice(c *ui.Context) {
	owner := ui.Box(c.Key("schedule-delete-notice-owner")).Absolute().Size(0, 0).PassThrough()
	seat := ui.Local(owner, "seat", func() scheduleDeleteNoticeSeat { return scheduleDeleteNoticeSeat{} })
	closeVisible := func() {
		if seat.showing {
			c.CloseToast(scheduleDeleteToastID)
			seat.showing = false
		}
	}
	s.mu.Lock()
	store := s.schedule
	s.mu.Unlock()
	if store == nil {
		closeVisible()
		return
	}
	notice, fresh := store.ConsumeDeletionNotice()
	if notice.Sequence == 0 {
		closeVisible()
		return
	}
	if !fresh {
		return
	}
	// The original reports only failure: a confirmed deletion is already visible
	// when the authoritative catalog removes its row, not another success banner.
	if notice.Outcome != "failed" {
		closeVisible()
		store.DismissDeletionNotice(notice)
		return
	}
	seat.showing = true
	c.AddToast(ui.Toast{ID: scheduleDeleteToastID, Description: i18n.T(i18n.KeyCmdCouldNotDeleteTheTask), Type: "warning", Timeout: 4 * time.Second,
		OnClose: func() { seat.showing = false; store.DismissDeletionNotice(notice) },
	})
}
