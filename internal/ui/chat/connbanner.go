package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// ConnBannerView paints a connection / mux status strip above the transcript.
// When onReconnect is set, a retry control is shown (Electron reconnect affordance).
func ConnBannerView(c *ui.Context, palette theme.Colors, text string, muted, field, border ui.Color, onReconnect func()) {
	text = normalizeConversationNotice(text)
	if text == "" {
		return
	}
	t := design.For(c)
	ui.Column(c).Background(t.Muted).Children(func() {
		ui.Row(c).FillWidth().Padding(design.Sp(1.5), design.Sp(3), design.Sp(1.5), design.Sp(3)).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, text).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.MutedForeground).Grow(1)
			if onReconnect != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyChatReconnect), buttonGhost, false).Clicked() {
					onReconnect()
				}
			}
		})
		coss.Separator(c, coss.Horizontal)
	})
}
