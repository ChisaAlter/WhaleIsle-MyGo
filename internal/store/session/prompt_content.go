package session

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/markdownsyntax"
)

// PromptContentBlocks builds session/prompt content parts from a draft that may
// mix text with markdown images / [image] placeholders (Electron InputBar).
func PromptContentBlocks(text string) []json.RawMessage {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	refs := ImageRefs(text)
	plain := text
	// Also strip inline markdown images from the text block when we emit image parts.
	if len(refs) > 0 {
		plain = strings.TrimSpace(markdownsyntax.WithoutImages(plain))
		plain = strings.Join(strings.Fields(plain), " ")
	}
	var out []json.RawMessage
	if plain != "" {
		block, _ := json.Marshal(map[string]string{"type": "text", "text": plain})
		out = append(out, block)
	}
	for _, ref := range refs {
		if id, ok := strings.CutPrefix(ref, "attachment:"); ok && id != "" {
			block, _ := json.Marshal(map[string]string{"type": "image", "attachmentId": id})
			out = append(out, block)
			continue
		}
		block, _ := json.Marshal(map[string]string{"type": "image", "path": ref})
		out = append(out, block)
	}
	if len(out) == 0 {
		block, _ := json.Marshal(map[string]string{"type": "text", "text": text})
		out = append(out, block)
	}
	return out
}
