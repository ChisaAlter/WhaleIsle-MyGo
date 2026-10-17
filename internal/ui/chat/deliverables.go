package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// DeliverablesView paints a compact deliverables/presented card.
// onOpen, when set, makes each path a button (open in files dock).
func DeliverablesView(c *ui.Context, msg session.Message, fg, muted, border, field ui.Color, onOpen func(path string)) {
	ui.Column(c).Gap(4).Background(field).Border(1, border).Radius(6).Padding(8, 10, 8, 10).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatDeliverables)).FontSize(11).Bold().TextColor(muted)
		ui.Box(c).Fill().Height(1).Background(border)
		files := msg.Files
		if len(files) == 0 && msg.Body != "" {
			files = []string{msg.Body}
		}
		for _, f := range files {
			path := f
			if onOpen != nil {
				if ui.Button(c, path).FontSize(12).BorderWidth(0).Background(ui.Transparent).
					TextColor(fg).FillWidth().Clicked() {
					onOpen(path)
				}
			} else {
				ui.Text(c, path).FontSize(12).TextColor(fg)
			}
		}
	})
}
