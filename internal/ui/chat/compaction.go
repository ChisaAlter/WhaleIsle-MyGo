package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// CompactionView paints a compact compaction/* system card.
func CompactionView(c *ui.Context, msg session.Message, fg, muted, field ui.Color) {
	ui.Column(c).Gap(2).Background(field).Radius(6).Padding(8, 10, 8, 10).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatCompaction)).FontSize(11).Bold().TextColor(muted)
		ui.Text(c, msg.Body).FontSize(12).TextColor(fg)
	})
}
