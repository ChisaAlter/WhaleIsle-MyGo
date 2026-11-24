package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
	remaining := notice.Expires.Sub(c.Now())
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
	t := tok(c)
	ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Padding(design.Sp(1), design.Sp(3), 0, design.Sp(3)).Children(func() {
		ui.Text(c, title+" · "+notice.Title).FontSize(fs(12)).TextColor(t.MutedForeground).Grow(1).MinWidth(0).SingleLine()
		label := workspaceText("撤销归档", "Undo archive")
		if coss.Button(c, label, coss.ButtonOptions{
			Variant: coss.ButtonOutline,
			Size:    coss.ButtonXS,
			Key:     "archive-undo-btn",
		}).Label(label).Shrink(0).Clicked() && m.OnUndoArchive != nil {
			m.OnUndoArchive(notice.Token)
		}
	})
}
