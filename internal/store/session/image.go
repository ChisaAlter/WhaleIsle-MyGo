package session

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"whaleisle/internal/markdownsyntax"
)

func imagePlaceholderLine(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return "[image] " + path
}

func looksLikeImagePath(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(s), "data:image/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(s))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico", ".tif", ".tiff", ".avif":
		return true
	default:
		return false
	}
}

func markdownImagePath(s string) string {
	refs := markdownsyntax.ImageReferences(s)
	if len(refs) == 0 {
		return ""
	}
	return refs[0]
}

func stringAsImagePlaceholder(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if refs, only := markdownsyntax.OnlyImages(s); only {
		lines := make([]string, 0, len(refs))
		for _, ref := range refs {
			lines = append(lines, imagePlaceholderLine(ref))
		}
		return strings.Join(lines, "\n")
	}
	if looksLikeImagePath(s) {
		return imagePlaceholderLine(s)
	}
	return ""
}

func imagePathFromBlock(raw json.RawMessage) string {
	var blk struct {
		Type         string `json:"type"`
		Path         string `json:"path"`
		URL          string `json:"url"`
		Src          string `json:"src"`
		AttachmentId string `json:"attachmentId"`
		Attachment   *struct {
			ID string `json:"attachmentId"`
		} `json:"attachment"`
	}
	if err := json.Unmarshal(raw, &blk); err != nil {
		return ""
	}
	if blk.Type != "" && blk.Type != "image" {
		return ""
	}
	if id := strings.TrimSpace(blk.AttachmentId); id != "" {
		return "attachment:" + id
	}
	if blk.Type == "image" && blk.Attachment != nil {
		if id := strings.TrimSpace(blk.Attachment.ID); id != "" {
			return "attachment:" + id
		}
	}
	for _, p := range []string{blk.Path, blk.URL, blk.Src} {
		p = strings.TrimSpace(p)
		if p != "" {
			return p
		}
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	for _, key := range []string{"path", "url", "src", "href", "file", "imageUrl", "imageURL"} {
		if v, ok := obj[key]; ok {
			if p := extractImagePathValue(v); p != "" {
				return p
			}
		}
	}
	return ""
}

func extractImagePathValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if looksLikeImagePath(s) {
			return s
		}
		if p := markdownImagePath(s); p != "" {
			return p
		}
		return ""
	}
	return imagePathFromBlock(raw)
}

func appendBlockText(out string, block json.RawMessage) string {
	if p := imagePathFromBlock(block); p != "" {
		line := imagePlaceholderLine(p)
		if out == "" {
			return line
		}
		return out + "\n" + line
	}
	text := extractTextish(block)
	if out != "" && text != "" {
		lines := strings.Split(out, "\n")
		if strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "[image] ") {
			return out + "\n" + text
		}
	}
	return out + text
}
