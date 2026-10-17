package frame

import (
	"github.com/egoist/mygo/ui"
	"time"
)

func archiveUndoToast(c *ui.Context, m Model) {
	const id = "session-archive-undo"
	if m.Store == nil {
		return
	}
	notice := m.Store.ArchiveUndoNotice()
	if notice == nil {
		c.CloseToast(id)
		return
	}
	remaining := time.Until(notice.Expires)
	if remaining <= 0 {
		c.CloseToast(id)
		return
	}
	c.After(remaining)
	if !notice.Announce {
		return
	}
	store, token := m.Store, notice.Token
	c.AddToast(ui.Toast{ID: id, Title: archiveText("会话已归档", "Session archived"), Description: notice.Title, Type: "success", Timeout: remaining, Action: archiveText("撤销归档", "Undo archive"), OnAction: func() { store.UndoArchive(token) }})
}
