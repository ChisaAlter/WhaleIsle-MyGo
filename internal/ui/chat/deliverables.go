package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// DeliverablesView paints a compact deliverables/presented card.
// onOpen, when set, makes each path a button (open in files dock).
func DeliverablesView(c *ui.Context, msg session.Message, _, _, _, _ ui.Color, onOpen func(path string)) {
	t := design.For(c)
	chatCardSurface(c, ui.Column(c).Gap(design.Sp(1)).Padding(design.Sp(2), design.Sp(2.5), design.Sp(2), design.Sp(2.5)), design.RadiusSM).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatDeliverables)).FontSize(11).Bold().TextColor(t.MutedForeground)
		coss.Separator(c, coss.Horizontal)
		files := msg.Files
		if len(files) == 0 && msg.Body != "" {
			files = []string{msg.Body}
		}
		for _, f := range files {
			path := f
			if onOpen != nil {
				if coss.Button(c, path, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).FillWidth().Justify(ui.Start).Clicked() {
					onOpen(path)
				}
			} else {
				ui.Text(c, path).FontSize(design.TextXS.Size).TextColor(t.CardForeground)
			}
		}
	})
}
