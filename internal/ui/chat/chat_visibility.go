package chat

import (
	"encoding/json"

	"whaleisle/internal/store/session"
)

type contextBlock struct {
	Type     string `json:"type"`
	ToolName string `json:"toolName"`
	Text     string `json:"text"`
}

func contextMessageParts(msg session.Message) (source map[string]json.RawMessage, content []contextBlock) {
	fields, blocks, _ := readContextBodyMessage(msg)
	if fields != nil {
		source = make(map[string]json.RawMessage, len(fields))
		for _, field := range fields {
			source[field.Name] = field.Raw
		}
	}
	for _, raw := range blocks {
		fields, _ := contextObjectEntries(raw)
		kind, _ := contextSourceString(fields, "type")
		name, _ := contextSourceString(fields, "toolName")
		text, _ := contextSourceString(fields, "text")
		content = append(content, contextBlock{Type: kind, ToolName: name, Text: text})
	}
	return source, content
}

// visibleChatMessages filters the main Chat order, never its retained snapshot.
// Tool changes stay visible even in mixed content; claimed nonhuman messages
// retain their independent Turn-trigger notice.
func visibleChatMessages(messages []session.Message) []session.Message {
	out := make([]session.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Kind == session.MessageKindSystemPrompt || msg.Kind == session.MessageKindCommand && msg.Command != nil && msg.Command.Name == "permission" {
			continue
		}
		if msg.Kind == session.MessageKindContext && !msg.ContextWaking {
			_, content := contextMessageParts(msg)
			toolChange := false
			for _, block := range content {
				if block.Type == "tool-addition" || block.Type == "tool-removal" {
					toolChange = true
					break
				}
			}
			if !toolChange {
				continue
			}
		}
		out = append(out, msg)
	}
	return out
}
