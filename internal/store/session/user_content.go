package session

import (
	"encoding/json"
	"strings"
)

// UserContent preserves durable content occurrences. Display names are not
// identities: two uploads with the same name remain two separate attachments.
type UserContent struct {
	Structured  bool
	Text        string
	Attachments []UserContentAttachment
	ExtraBlocks []json.RawMessage
}

type UserContentAttachment struct {
	Type         string
	AttachmentID string
	Name         string
	MediaType    string
	Bytes        float64
}

// UserMessageContent is shared by admission to the transcript and its renderer.
// Unknown content stays opaque JSON rather than disappearing into text extraction.
func UserMessageContent(msg Message) UserContent {
	var event struct {
		Data struct {
			Content json.RawMessage `json:"content"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"data"`
	}
	if len(msg.Raw) == 0 || json.Unmarshal(msg.Raw, &event) != nil {
		return UserContent{Text: msg.Body}
	}
	raw := event.Data.Content
	if len(raw) == 0 {
		raw = event.Data.Message.Content
	}
	var blocks []json.RawMessage
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &blocks) != nil {
		return UserContent{Text: msg.Body}
	}
	out := UserContent{Structured: true}
	var text strings.Builder
	for _, raw := range blocks {
		var block struct {
			Type       string  `json:"type"`
			Text       *string `json:"text"`
			Attachment *struct {
				AttachmentID string  `json:"attachmentId"`
				Name         string  `json:"name"`
				MediaType    string  `json:"mediaType"`
				Bytes        float64 `json:"bytes"`
			} `json:"attachment"`
		}
		if json.Unmarshal(raw, &block) == nil {
			if block.Type == "text" && block.Text != nil {
				text.WriteString(*block.Text)
				continue
			}
			if (block.Type == "image" || block.Type == "file" || block.Type == "attachment") && block.Attachment != nil {
				a := block.Attachment
				kind := block.Type
				if kind == "attachment" {
					kind = "file"
				}
				out.Attachments = append(out.Attachments, UserContentAttachment{Type: kind, AttachmentID: a.AttachmentID, Name: a.Name, MediaType: a.MediaType, Bytes: a.Bytes})
				continue
			}
		}
		out.ExtraBlocks = append(out.ExtraBlocks, append(json.RawMessage(nil), raw...))
	}
	out.Text = text.String()
	return out
}
