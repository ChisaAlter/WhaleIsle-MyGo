package main

import (
	"time"

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
	text, tone := resumeText("任务已删除", "Task deleted."), "success"
	if notice.Outcome == "failed" {
		text, tone = resumeText("无法删除任务", "Could not delete the task."), "warning"
	}
	c.AddToast(ui.Toast{ID: scheduleDeleteToastID, Description: text, Type: tone, Timeout: 4 * time.Second,
		OnClose: func() { store.DismissDeletionNotice(notice) },
	})
}
