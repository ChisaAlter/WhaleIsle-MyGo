package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
)

func paintPendingSubmission(c *ui.Context, p session.PendingSubmission, fg, muted ui.Color) {
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		if p.Text != "" {
			ui.Text(c, p.Text).FontSize(13).TextColor(fg)
		}
		for _, a := range p.Attachments {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if (a.ImageData != "" || strings.HasPrefix(a.MediaType, "image/")) && a.Path != "" {
					if bmp := loadChatBitmap(a.Path); bmp != nil {
						ui.Image(c, bmp).Width(64).Height(48).Fit(ui.Contain)
					}
				}
				name := a.Name
				if name == "" {
					name = a.Path
				}
				ui.Text(c, name).FontSize(11).TextColor(muted)
			})
		}
		status := "发送中 / Sending…"
		if p.Accepted {
			status = "已接受，等待会话更新 / Accepted, waiting for conversation update"
		}
		if p.Placement == "steering" {
			status = "插话 · " + status
		}
		ui.Text(c, status).FontSize(11).TextColor(muted)
	})
}
