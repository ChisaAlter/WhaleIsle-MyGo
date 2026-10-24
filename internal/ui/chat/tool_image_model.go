package chat

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"whaleisle/internal/store/session"
)

// toolImageModel is pure presentation data from the settled result's image
// blocks. Paths and provider-owned attachment IDs retain their recorded spelling;
// neither grants filesystem, attachment-fetch, or host-opening authority.
// Measures retain JavaScript Number.isInteger semantics beyond native int bounds.
type toolImageModel struct {
	Path      string
	HasPath   bool
	Images    []toolImageAttachment
	ExtraText *string
}

type toolImageAttachment struct {
	AttachmentID, MediaType string
	Bytes, Width, Height    float64
	Name                    *string
	OriginalDimensions      *toolImageDimensions
}

type toolImageDimensions struct {
	Width, Height float64
}

// toolImageModelFor follows the installed read_image card. Root results require
// a retained raw call head; nested receipts own their settled name and arguments.
// Content owns every image reference, while metadata contributes only the path.
func toolImageModelFor(call *session.ToolCallView) *toolImageModel {
	if call == nil || call.Result == nil || call.State != "ok" {
		return nil
	}
	result, ok := terminalModelReadResult(call.Result)
	if !ok || result.failed {
		return nil
	}
	content, ok := contextRawArray(result.content)
	if !ok {
		return nil
	}
	// The installed childResult embeds its call even when the start event is
	// outside the retained window. Root results instead borrow a retained head.
	var data, args map[string]json.RawMessage
	hasParent := false
	if result.nested {
		envelope, _ := terminalModelObject(call.Result.Raw)
		data, _ = terminalModelObject(envelope["data"])
		_, hasParent = data["parentCallId"]
		args, ok = terminalModelObject(data["arguments"])
	} else {
		if call.Call == nil {
			return nil
		}
		head, valid := terminalModelObject(call.Call.Raw)
		if !valid {
			return nil
		}
		typ, valid := terminalModelString(head["type"])
		if !valid || typ != "tool/call" || call.Call.EventType != "" && call.Call.EventType != typ {
			return nil
		}
		data, ok = terminalModelObject(head["data"])
		if !ok {
			return nil
		}
		var raw string
		raw, ok = terminalModelString(data["arguments"])
		if ok {
			args, ok = terminalModelObject(json.RawMessage(raw))
		}
	}
	name, _ := terminalModelString(data["name"])
	if !ok || name != "read_image" {
		return nil
	}
	filePath, ok := terminalModelString(args["file_path"])
	if !ok || terminalModelBlank(filePath) {
		return nil
	}
	meta, _ := terminalModelObject(result.meta)
	path, validPath := terminalModelString(meta["path"])
	if !validPath || path == "" {
		if !hasParent {
			return nil
		}
		path = filePath
	}
	parts := make([]string, 0, len(content))
	hasEnvelope := false
	for _, raw := range content {
		part, valid := terminalModelObject(raw)
		if !valid {
			return nil
		}
		typ, _ := terminalModelString(part["type"])
		switch typ {
		case "image":
		case "text":
			text, valid := terminalModelString(part["text"])
			if !valid {
				return nil
			}
			if toolImageModelEnvelope.MatchString(text) {
				hasEnvelope = true
			} else {
				parts = append(parts, text)
			}
		default:
			return nil
		}
	}
	if !hasEnvelope {
		return nil
	}
	images := toolImageModelReferences(content)
	if images == nil {
		return nil
	}
	model := &toolImageModel{Path: path, HasPath: true, Images: images}
	if len(parts) != 0 {
		text := strings.Join(parts, "\n")
		model.ExtraText = &text
	}
	return model
}

// toolGenericImageModelFor follows the installed generic image card for any
// successful settled tool, including result-only history windows. The caller
// chooses priority relative to specialized cards; metadata and arguments do not
// participate. Unknown objects and arrays remain visible as formatted JSON.
func toolGenericImageModelFor(call *session.ToolCallView) *toolImageModel {
	if call == nil || call.Result == nil || call.State != "ok" {
		return nil
	}
	result, ok := terminalModelReadResult(call.Result)
	if !ok || result.failed {
		return nil
	}
	content, ok := contextRawArray(result.content)
	if !ok {
		return nil
	}
	images := toolImageModelReferences(content)
	if images == nil {
		return nil
	}
	parts := make([]string, 0, len(content))
	for _, raw := range content {
		part, object := terminalModelObject(raw)
		if !object {
			// Arrays are JavaScript objects too; null and scalar blocks are
			// ignored. contextRawArray already validated each JSON value.
			if trimmed := bytes.TrimSpace(raw); len(trimmed) != 0 && trimmed[0] == '[' {
				parts = append(parts, contextFormattedJSON(raw))
			}
			continue
		}
		typ, _ := terminalModelString(part["type"])
		if typ == "image" {
			continue
		}
		if text, valid := terminalModelString(part["text"]); typ == "text" && valid {
			parts = append(parts, text)
		} else {
			parts = append(parts, contextFormattedJSON(raw))
		}
	}
	model := &toolImageModel{Images: images}
	if len(parts) != 0 {
		text := strings.Join(parts, "\n")
		model.ExtraText = &text
	}
	return model
}

func toolImageModelReferences(content []json.RawMessage) []toolImageAttachment {
	var images []toolImageAttachment
	for _, raw := range content {
		part, _ := terminalModelObject(raw)
		if typ, _ := terminalModelString(part["type"]); typ != "image" {
			continue
		}
		attachment, ok := terminalModelObject(part["attachment"])
		if !ok {
			return nil
		}
		id, validID := terminalModelString(attachment["attachmentId"])
		media, validMedia := terminalModelString(attachment["mediaType"])
		if !validID || id == "" || !validMedia {
			return nil
		}
		switch media {
		case "image/png", "image/jpeg", "image/webp", "image/gif":
		default:
			return nil
		}
		byteCount, validBytes := toolReadModelInteger(attachment["bytes"], 1)
		width, validWidth := toolReadModelInteger(attachment["width"], 1)
		height, validHeight := toolReadModelInteger(attachment["height"], 1)
		if !validBytes || !validWidth || !validHeight {
			return nil
		}
		image := toolImageAttachment{AttachmentID: id, MediaType: media, Bytes: byteCount, Width: width, Height: height}
		if rawName, exists := attachment["name"]; exists {
			name, valid := terminalModelString(rawName)
			if !valid {
				return nil
			}
			image.Name = &name
		}
		if rawDimensions, exists := attachment["originalDimensions"]; exists {
			dimensions, valid := terminalModelObject(rawDimensions)
			if !valid {
				return nil
			}
			width, validWidth := toolReadModelInteger(dimensions["width"], 1)
			height, validHeight := toolReadModelInteger(dimensions["height"], 1)
			if !validWidth || !validHeight {
				return nil
			}
			image.OriginalDimensions = &toolImageDimensions{Width: width, Height: height}
		}
		images = append(images, image)
	}
	return images
}

// Structural separators are LF, the path excludes LF, and the closing tag must
// be at the true end. CR and Unicode line separators remain valid within paths
// and bodies, exactly as /^...$/u in the installed JavaScript model.
var toolImageModelEnvelope = regexp.MustCompile(`\A<path>[^\n]*</path>\n<type>image</type>\n<content>\n(?s:.*)\n</content>\z`)
