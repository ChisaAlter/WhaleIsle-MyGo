package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// The installed session-log-export action owns this menu. View selection,
// plugin discovery and settings have separate entry points.
func conversationMoreMenu(c *ui.Context, m Model, snapshot session.ViewState, ink ui.Color) {
	id := snapshot.SelectedID
	if id == "" || conversationIsManaged(snapshot) || (m.OnDownloadSessionLog == nil && m.OnOpenSessionFeedback == nil) {
		return
	}
	button := conversationHeaderButton(c, "conversation-more", paneLabel("更多操作", "More actions"), "EllipsisOutline", true, ink, func() {}, 0)
	button.Menu(func(menu *ui.Menu) {
		current := m.Store != nil && m.Store.SelectedSessionID() == id
		busy := m.SessionLogDownloading != nil && m.SessionLogDownloading(id)
		if m.OnDownloadSessionLog != nil && menu.Item(paneLabel("下载会话日志", "Download session log")).Disabled(!current || busy).Chosen() {
			m.OnDownloadSessionLog(id)
		}
		if m.OnOpenSessionFeedback != nil && menu.Item(paneLabel("反馈", "Feedback")).Disabled(!current).Chosen() {
			m.OnOpenSessionFeedback(id)
		}
	})
}
