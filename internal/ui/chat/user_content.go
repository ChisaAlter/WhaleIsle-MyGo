package chat

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

type userContentMemo struct {
	raw   string
	value session.UserContent
}

func (m *userContentMemo) resolve(msg session.Message) {
	if m.raw == string(msg.Raw) && m.value.Structured {
		return
	}
	m.raw = string(msg.Raw)
	m.value = session.UserMessageContent(msg)
}

func userContentAttachments(c *ui.Context, attachments []session.UserContentAttachment, fg, muted, border, field ui.Color, open func(string)) {
	if len(attachments) == 0 {
		return
	}
	ui.Row(c.Key("attachments")).Wrap().Gap(design.Sp(2)).Justify(ui.End).MaxWidthPercent(100).Children(func() {
		for index, attachment := range attachments {
			child := c.Key(fmt.Sprintf("attachment:%d:%s", index, attachment.AttachmentID))
			if attachment.Type == "image" {
				ref := "attachment:" + attachment.AttachmentID
				if len(attachments) > 1 {
					paintImageTile(child, ref, muted, open)
				} else {
					ImageInlineView(child, ref, fg, muted, open)
				}
			} else {
				attachmentCardView(child, userFileAttachment{AttachmentID: attachment.AttachmentID, Name: attachment.Name, MediaType: attachment.MediaType, Bytes: attachment.Bytes}, fg, muted, border, field)
			}
		}
	})
}

// Unknown blocks are inspectable evidence, never guessed executable UI or links.
// Masking changes display only; Copy preserves exactly the durable JSON block.
func userExtraBlockView(c *ui.Context, raw json.RawMessage, fg, muted ui.Color) {
	t := design.For(c)
	root := ui.Column(c.Key("extra-content")).FillWidth().MinWidth(0).Gap(design.Sp(1))
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		label := referenceCopy("Additional content", "附加内容")
		chevron := coss.IconChevronDown
		if *open {
			chevron = coss.IconChevronUp
		}
		button := coss.Button(c, label, coss.ButtonOptions{
			Variant:      coss.ButtonGhost,
			Size:         coss.ButtonXS,
			TrailingIcon: chevron,
		}).Role(ui.RoleDisclosure).Expanded(*open).FillWidth().MinWidth(0)
		if button.Clicked() {
			*open = !*open
		}
		if !*open {
			return
		}
		var formatted bytes.Buffer
		body := string(raw)
		if json.Indent(&formatted, raw, "", "  ") == nil {
			body = formatted.String()
		}
		ui.Text(c, maskedUserText(body)).Font(theme.RenderPreferences().Code.Family).FontSize(design.TextXS.Size).TextColor(t.Foreground).Selectable().MinWidth(0)
		if coss.Button(c, referenceCopy("Copy content", "复制内容"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Icon: coss.IconCopy}).Clicked() {
			c.Services().WriteClipboard(string(raw))
		}
	})
}
