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
	var event struct {
		Type string `json:"type"`
		Data struct {
			Source  map[string]json.RawMessage `json:"source"`
			Content []contextBlock             `json:"content"`
			Message *struct {
				Source  map[string]json.RawMessage `json:"source"`
				Content []contextBlock             `json:"content"`
			} `json:"message"`
		} `json:"data"`
	}
	if json.Unmarshal(msg.Raw, &event) != nil {
		return nil, nil
	}
	if event.Type == "developer/message" && event.Data.Message != nil {
		return event.Data.Message.Source, event.Data.Message.Content
	}
	return event.Data.Source, event.Data.Content
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
