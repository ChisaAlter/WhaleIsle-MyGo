package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// ConnBannerView paints a connection / mux status strip above the transcript.
// When onReconnect is set, a retry control is shown (Electron reconnect affordance).
func ConnBannerView(c *ui.Context, palette theme.Colors, text string, muted, field, border ui.Color, onReconnect func()) {
	if text == "" {
		return
	}
	ui.Column(c).Background(field).Children(func() {
		ui.Row(c).FillWidth().Padding(6, 12, 6, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, text).FontSize(12).TextColor(muted).Grow(1)
			if onReconnect != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyChatReconnect), buttonGhost, false).Clicked() {
					onReconnect()
				}
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)
	})
}
