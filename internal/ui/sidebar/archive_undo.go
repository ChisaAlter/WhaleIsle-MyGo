package sidebar

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// archiveUndoRow keeps an archived session's undo affordance in the surface the
// session lived in: a muted receipt line above the sidebar footer carrying a
// real 撤销 button, for the store's undo window instead of a floating toast.
// The store owns the expiry; c.After schedules the repaint that retires it.
func archiveUndoRow(c *ui.Context, m Model) {
	notice := m.ArchiveUndo
	if notice == nil || notice.Expires.IsZero() {
		return
	}
	remaining := time.Until(notice.Expires)
	if remaining <= 0 {
		return
	}
	c.After(remaining)
	title := workspaceText("会话已归档", "Session archived")
	if notice.Stopped {
		title = workspaceText("已停止并归档", "Stopped and archived")
	}
	if notice.Announce {
		c.Announce(title)
	}
	muted := c.Theme().TextMuted
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(4, 12, 0, 12).Children(func() {
		ui.Text(c, title+" · "+notice.Title).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
		label := workspaceText("撤销归档", "Undo archive")
		if ui.Button(c, label).Label(label).Height(24).Padding(0, 8).FontSize(theme.InterfaceSize(12)).Shrink(0).Clicked() && m.OnUndoArchive != nil {
			m.OnUndoArchive(notice.Token)
		}
	})
}
