package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chat"
)

func sessionFeedbackDialog(c *ui.Context, m Model) {
	chat.SessionFeedbackDialog(c, m.Store, m.ThemeID, m.SystemDark)
}
