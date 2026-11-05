package chat

import (
	"fmt"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Original MessageItem.module.css: the file attachment row renders a 240×64
// card with a 28 DIP kind glyph, the name on 14/22 and an "EXT · size" line.
const (
	attachmentCardWidth  = float32(240)
	attachmentCardHeight = float32(64)
	attachmentGlyphSize  = float32(28)
	attachmentGlyphSeat  = float32(28)
)

// userFileAttachment is one durable file part of a user message.
type userFileAttachment struct {
	AttachmentID string
	Name         string
	MediaType    string
	Bytes        float64
}

// userFileAttachments reads the structured file parts of a user/message event.
// The Host keeps attachment metadata in the durable content, which is the only
// place a transcript can learn a file's name, kind and size; bodies that were
// written before this projection carry text only, so no card is invented.
func userFileAttachments(msg session.Message) []userFileAttachment {
	var out []userFileAttachment
	for _, a := range session.UserMessageContent(msg).Attachments {
		if a.Type != "file" {
			continue
		}
		name := a.Name
		if name == "" {
			name = a.AttachmentID
		}
		out = append(out, userFileAttachment{AttachmentID: a.AttachmentID, Name: name, MediaType: a.MediaType, Bytes: a.Bytes})
	}
	return out
}

type attachmentContentPart struct {
	Type       string `json:"type"`
	Attachment *struct {
		AttachmentID string  `json:"attachmentId"`
		Name         string  `json:"name"`
		MediaType    string  `json:"mediaType"`
		Bytes        float64 `json:"bytes"`
	} `json:"attachment"`
}

// attachmentMeta is the "EXT · size" line; a name without either part shows
// the media type's subtype so the card never reads empty.
func attachmentMeta(name, mediaType string, bytes float64) string {
	fields := make([]string, 0, 2)
	if ext := attachmentExtension(name); ext != "" {
		fields = append(fields, ext)
	} else if subtype := mediaSubtype(mediaType); subtype != "" {
		fields = append(fields, subtype)
	}
	if size := formatAttachmentSize(bytes); size != "" {
		fields = append(fields, size)
	}
	return strings.Join(fields, " · ")
}

func attachmentExtension(name string) string {
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(strings.TrimSpace(name))), ".")
	if len(ext) > 8 {
		return ""
	}
	return ext
}

func mediaSubtype(mediaType string) string {
	if slash := strings.IndexByte(mediaType, '/'); slash >= 0 && slash+1 < len(mediaType) {
		return mediaType[slash+1:]
	}
	return ""
}

// formatAttachmentSize follows the Original formatBytes: B, KB, MB, GB with one
// decimal under ten followed by the unit, and no decimal above it.
func formatAttachmentSize(bytes float64) string {
	if bytes <= 0 {
		return ""
	}
	if bytes < 1024 {
		return fmt.Sprintf("%d B", int(bytes))
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := bytes / 1024
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if value >= 10 {
		return fmt.Sprintf("%.0f %s", value, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

// attachmentKindIcon picks the leading glyph by extension, then media type.
func attachmentKindIcon(name, mediaType string) *ui.SVG {
	ext := attachmentExtension(name)
	switch ext {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "svg", "avif", "heic":
		return attachmentIconImage
	case "pdf":
		return attachmentIconDocument
	case "doc", "docx", "rtf", "odt", "txt", "md", "log":
		return attachmentIconText
	case "xls", "xlsx", "csv", "tsv", "ods":
		return attachmentIconSheet
	case "zip", "tar", "gz", "tgz", "7z", "rar", "bz2", "xz":
		return attachmentIconArchive
	case "go", "ts", "tsx", "js", "jsx", "py", "rs", "java", "c", "h", "cpp", "hpp", "cs", "rb", "php", "sh", "json", "yaml", "yml", "toml", "sql", "css", "html", "xml":
		return attachmentIconCode
	case "mp3", "wav", "m4a", "flac", "ogg":
		return attachmentIconAudio
	case "mp4", "mov", "webm", "mkv", "avi":
		return attachmentIconVideo
	}
	switch {
	case strings.HasPrefix(mediaType, "image/"):
		return attachmentIconImage
	case strings.HasPrefix(mediaType, "audio/"):
		return attachmentIconAudio
	case strings.HasPrefix(mediaType, "video/"):
		return attachmentIconVideo
	case strings.HasPrefix(mediaType, "text/"):
		return attachmentIconText
	}
	return attachmentIconFile
}

// attachmentCardView paints one file part of a user message. The Original card
// is display-only: it names the file and its size, and the transcript's file
// mentions (not the card) are what open anything.
func attachmentCardView(c *ui.Context, file userFileAttachment, fg, muted, border, field ui.Color) {
	name := strings.TrimSpace(file.Name)
	if name == "" {
		name = i18n.T(i18n.KeyChatAttachmentFile)
	}
	meta := attachmentMeta(name, file.MediaType, file.Bytes)
	icon := attachmentKindIcon(name, file.MediaType)
	ui.Row(c).Label("Attachment "+name).Width(attachmentCardWidth).MinHeight(attachmentCardHeight).
		Radius(20).Border(0.5, border).Padding(0, 12, 0, 12).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(attachmentGlyphSize, attachmentGlyphSize).Center().Children(func() {
			ui.Icon(c, icon).Width(attachmentGlyphSeat).Height(attachmentGlyphSeat).TextColor(muted)
		})
		ui.Column(c).MinWidth(0).Grow(1).Gap(1).Children(func() {
			ui.Text(c, name).FontSize(14).FixedLineHeight(22).FontWeight(500).TextColor(fg).SingleLine()
			if meta != "" {
				ui.Text(c, meta).FontSize(12).FixedLineHeight(15).TextColor(muted).SingleLine()
			}
		})
	})
}

// pendingAttachmentCard adapts a staged composer attachment to the same card so
// an optimistic echo and the durable message look alike.
func pendingAttachmentCard(a session.ComposerAttachment) userFileAttachment {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		name = strings.TrimSpace(a.Path)
	}
	bytes := a.Upload.File.Bytes
	return userFileAttachment{AttachmentID: a.Upload.File.AttachmentId, Name: name, MediaType: a.MediaType, Bytes: bytes}
}

// isImageAttachment reports whether a staged attachment paints as an image.
func isImageAttachment(a session.ComposerAttachment) bool {
	return a.ImageData != "" || strings.HasPrefix(strings.ToLower(a.MediaType), "image/")
}
