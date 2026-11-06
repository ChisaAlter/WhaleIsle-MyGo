package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// paintPendingSubmission draws the optimistic echo of a just-sent prompt. It
// reuses the durable bubble and attachment cards so the message does not change
// shape when its record lands (Original MessageItem data-submission-echo).
func paintPendingSubmission(c *ui.Context, p session.PendingSubmission, fg, muted ui.Color) {
	border, field := muted.Alpha(0.28), muted.Alpha(0.08)
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.End).Children(func() {
		var files []session.ComposerAttachment
		for _, a := range p.Attachments {
			if !isImageAttachment(a) {
				files = append(files, a)
			}
		}
		var images []session.ComposerAttachment
		for _, a := range p.Attachments {
			if isImageAttachment(a) && strings.TrimSpace(a.Path) != "" {
				images = append(images, a)
			}
		}
		for _, a := range files {
			attachmentCardView(c, pendingAttachmentCard(a), fg, muted, border, field)
		}
		if len(images) > 0 {
			ui.Row(c).Wrap().Gap(8).Justify(ui.End).MaxWidthPercent(100).Children(func() {
				for _, a := range images {
					paintImageTile(c, a.Path, muted, nil)
				}
			})
		}
		if text := strings.TrimSpace(p.Text); text != "" {
			ui.Column(c).Label("User message bubble").MinWidth(0).MaxWidthPercent(100).
				Background(fg.Alpha(0.06)).Radius(20).Padding(10, 16, 10, 16).Children(func() {
				ui.Text(c, maskedUserText(text)).FontSize(13).FixedLineHeight(22).TextColor(fg)
			})
		}
		status := i18n.T(i18n.KeyChatPendingSending)
		if p.Accepted {
			status = i18n.T(i18n.KeyChatPendingAccepted)
		}
		if p.Placement == "steering" {
			status = i18n.T(i18n.KeyChatPendingSteering) + " · " + status
		}
		ui.Text(c, status).FontSize(11).TextColor(muted)
	})
}
