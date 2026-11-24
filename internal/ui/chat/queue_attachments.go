package chat

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

var queueImageLoads attachmentLoadCache

func queueAttachmentLabel(a session.QueueAttachment) string {
	name := a.Name
	if name == "" {
		name = a.AttachmentID
	}
	if a.Kind == "image" {
		return fmt.Sprintf("%s · %.0f×%.0f · %s", name, a.Width, a.Height, queueFileSize(a.Bytes))
	}
	return name + " · " + queueFileSize(a.Bytes)
}
func queueFileSize(bytes float64) string {
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", bytes/(1024*1024))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%.1f KB", bytes/1024)
	}
	return fmt.Sprintf("%.0f B", bytes)
}
func paintQueueAttachments(c *ui.Context, m PlanBoardModel, item session.PlanQueueItem, fg, muted ui.Color) {
	t := design.For(c)
	fg, muted = t.Foreground, t.MutedForeground
	for _, a := range item.Attachments {
		ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
			if a.Kind == "image" && m.LoadQueueImage != nil {
				key := m.Board.QueueActions.Scope + "\x00" + a.AttachmentID
				read := attachmentImageRead()
				state := queueImageLoads.ensure(key, func(ctx context.Context, _ string) (string, error) {
					path, err := m.LoadQueueImage(ctx, a.AttachmentID)
					if err == nil && loadFetchedChatBitmap(path) == nil {
						return "", fmt.Errorf("invalid or unreadable image")
					}
					return path, err
				}, c.Services().Invalidate)
				if bmp, _ := cachedChatBitmap(state.path, c.Services().Invalidate); bmp != nil {
					ui.Image(c, bmp).Width(64).Height(48).Fit(ui.Contain)
				} else {
					ui.Text(c, "[image]").Width(64).FontSize(11).TextColor(muted)
					if state.err != nil && coss.Button(c, i18n.T(i18n.KeyChatImageRetry), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
						if read.Key != "" {
							read.Forget(a.AttachmentID)
						}
						queueImageLoads.retry(key)
						c.Services().Invalidate()
					}
				}
			}
			ui.Text(c, queueAttachmentLabel(a)).FontSize(11).TextColor(fg)
		})
	}
}
