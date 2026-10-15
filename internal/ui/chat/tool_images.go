package chat

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
)

// ToolImageGalleryOptions keeps disclosure state in the owning chat model.
type ToolImageGalleryOptions struct {
	// TextOnly is an addressed read-only seat: never load images/open links,
	// and retain image attachment evidence in expanded tool output.
	TextOnly  bool
	ID        string
	Expanded  *map[string]bool
	OnOpen    func(string)
	OnOpenURL func(string)
}

func toolImageGallery(c *ui.Context, refs []string, st ToolCardStyle, opts ToolImageGalleryOptions) {
	key := "tool-images:" + opts.ID
	open := isThinkingExpanded(opts.Expanded, key)
	arrow := "▸"
	if open {
		arrow = "▾"
	}
	if ui.Button(c, fmt.Sprintf("%s 图片 / Images (%d)", arrow, len(refs))).Clicked() {
		toggleThinkingExpanded(opts.Expanded, key)
	}
	if open {
		for _, ref := range refs {
			ImageInlineView(c, ref, st.FG, st.Muted, opts.OnOpen)
		}
	}
}

// Content is the only authority for durable references. A single invalid image
// declines the entire gallery rather than showing an incomplete tool result.
func toolImageReferences(content []json.RawMessage) []string {
	var refs []string
	for _, raw := range content {
		var block map[string]json.RawMessage
		if json.Unmarshal(raw, &block) != nil {
			continue
		}
		var kind string
		_ = json.Unmarshal(block["type"], &kind)
		if kind != "image" {
			continue
		}
		var a map[string]json.RawMessage
		if json.Unmarshal(block["attachment"], &a) != nil || a == nil {
			return nil
		}
		var id, media string
		if json.Unmarshal(a["attachmentId"], &id) != nil || strings.TrimSpace(id) == "" {
			return nil
		}
		if json.Unmarshal(a["mediaType"], &media) != nil {
			return nil
		}
		switch media {
		case "image/png", "image/jpeg", "image/gif", "image/webp":
		default:
			return nil
		}
		for _, key := range []string{"bytes", "width", "height"} {
			if !positiveImageMeasure(a[key]) {
				return nil
			}
		}
		if name, ok := a["name"]; ok {
			var value string
			if string(name) == "null" || json.Unmarshal(name, &value) != nil {
				return nil
			}
		}
		if dims, ok := a["originalDimensions"]; ok {
			var d map[string]json.RawMessage
			if json.Unmarshal(dims, &d) != nil || !positiveImageMeasure(d["width"]) || !positiveImageMeasure(d["height"]) {
				return nil
			}
		}
		refs = append(refs, "attachment:"+id)
	}
	return refs
}

func positiveImageMeasure(raw json.RawMessage) bool {
	var n float64
	return json.Unmarshal(raw, &n) == nil && n > 0 && math.Trunc(n) == n
}

// Leave extension/unknown blocks visible; only gallery-owned images disappear
// from the textual evidence. Read-image envelope/path text is preserved intact.
func toolImageText(content []json.RawMessage, gallery bool) string {
	var parts []string
	for _, raw := range content {
		var block struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(raw, &block)
		if gallery && block.Type == "image" {
			continue
		}
		if text := extractLooseText(raw); text != "" {
			parts = append(parts, text)
		} else {
			parts = append(parts, string(raw))
		}
	}
	return strings.Join(parts, "\n")
}
