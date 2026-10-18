package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// CompactionView paints a compact compaction/* system card.
func CompactionView(c *ui.Context, msg session.Message, fg, muted, field ui.Color) {
	ui.Column(c).Gap(2).Background(field).Radius(6).Padding(8, 10, 8, 10).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatCompaction)).FontSize(11).Bold().TextColor(muted)
		if compact := msg.AutomaticCompaction; compact != nil {
			if compact.ShadowedItemCount != nil && compact.ShadowedTokenCount != nil {
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyChatCompactionCounts), *compact.ShadowedItemCount, *compact.ShadowedTokenCount)).FontSize(11).TextColor(muted)
			}
			if compact.Summary == "" {
				ui.Text(c, i18n.T(i18n.KeyChatCompactionSummaryMissing)).FontSize(12).TextColor(muted)
			} else {
				ui.Text(c, compact.Summary).FontSize(12).TextColor(fg).Selectable()
			}
		} else {
			ui.Text(c, msg.Body).FontSize(12).TextColor(fg)
		}
	})
}
