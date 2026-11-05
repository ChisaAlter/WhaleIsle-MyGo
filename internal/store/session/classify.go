package session

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh/events"
)

func classifyPersistenceEvent(raw json.RawMessage) (Message, bool) {
	// Prompt events have their own bounded disclosure projection. Never route
	// system text or request diagnostics through the generic text fallback.
	var promptHead struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &promptHead) == nil {
		if promptHead.Type == "system/message" || promptHead.Type == "request/header" {
			return Message{}, false
		}
		if promptHead.Type == "agent/inbox/spliced" {
			if msg, ok := classifyGroupLateCarrier(raw); ok {
				return msg, true
			}
		}
		if promptHead.Type == "assistant/message" || promptHead.Type == "assistant/attempt" {
			if msg, ok := classifyAssistantPersistence(raw); ok {
				return msg, true
			}
		}
		if promptHead.Type == "tool/call" || promptHead.Type == "assistant/message" {
			if msg, ok := classifyPersistenceEventTyped(raw); ok {
				return msg, true
			}
		}
	}
	return classifyPersistenceEventRaw(raw)
}

func classifyPersistenceEventTyped(raw json.RawMessage) (Message, bool) {
	v, err := events.UnmarshalEvent(raw)
	if err != nil {
		return Message{}, false
	}
	rawCopy := append(json.RawMessage(nil), raw...)
	switch ev := v.(type) {
	case events.ToolCall:
		if ev.Data.Name == "" {
			return Message{}, false
		}
		body := ev.Data.Name
		args := strings.TrimSpace(ev.Data.Arguments)
		if args != "" {
			body += " " + truncateRunes(args, 80)
		}
		return Message{
			Kind:      MessageKindTool,
			Role:      "tool",
			Seq:       ev.Seq,
			EventType: ev.Type,
			Raw:       rawCopy,
			Body:      body,
			Args:      prettyJSONOrRaw(args),
			EventTime: ev.Time,
		}, true
	case events.AssistantMessage:
		dataRaw, err := json.Marshal(ev.Data)
		if err != nil {
			return Message{}, false
		}
		body := extractTextish(dataRaw)
		if body == "" {
			return Message{}, false
		}
		msg := Message{
			Kind:      MessageKindText,
			Role:      "assistant",
			Seq:       ev.Seq,
			Body:      body,
			Raw:       rawCopy,
			Model:     ev.Data.Message.Source.Model,
			EventTime: ev.Time,
		}
		if id := ev.Data.Message.Id; id != "" {
			msg.ID = id
		}
		return msg, true
	default:
		return Message{}, false
	}
}

func classifyPersistenceEventRaw(raw json.RawMessage) (Message, bool) {
	var ev struct {
		Type string          `json:"type"`
		Seq  float64         `json:"seq"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil || ev.Type == "" {
		return Message{}, false
	}
	if ev.Type == "command/run" || ev.Type == "command/done" {
		return classifyCommandEvent(raw)
	}
	if ev.Type == "user/message" {
		if checkpoint, ok := classifyManualCompactionCheckpoint(raw); ok {
			return checkpoint, true
		}
		if msg, ok := classifyAutomaticCompactionCheckpoint(raw); ok {
			return msg, true
		}
		if msg, ok := classifyQuestionReply(raw); ok {
			return msg, true
		}
	}
	if ev.Type == "user/message" || ev.Type == "developer/message" {
		if msg, ok := classifyContextMessage(raw); ok {
			return msg, true
		}
	}
	if ev.Type == "turn/end" {
		return Message{Kind: MessageKindText, Role: "status", Seq: ev.Seq, EventType: ev.Type, Raw: append(json.RawMessage(nil), raw...), Body: "0"}, true
	}
	if isToolPersistenceType(ev.Type) {
		msg := Message{
			Kind:      MessageKindTool,
			Role:      "tool",
			Seq:       ev.Seq,
			EventType: ev.Type,
			Raw:       append(json.RawMessage(nil), raw...),
			Body:      toolEventSummary(ev.Type, ev.Data),
		}
		if ev.Type == "tool/call" {
			var d struct {
				Arguments string `json:"arguments"`
			}
			if json.Unmarshal(ev.Data, &d) == nil {
				msg.Args = prettyJSONOrRaw(d.Arguments)
			}
		} else if ev.Type == "tool/result" {
			msg.Args = prettyJSONOrRaw(string(ev.Data))
			msg.Web = parseToolWebResult(ev.Data)
		}
		return msg, true
	}
	if ev.Type == "workspace/changes" {
		return Message{Kind: MessageKindTurnDelivery, Role: "assistant", Seq: ev.Seq, EventType: ev.Type, Raw: append(json.RawMessage(nil), raw...)}, true
	}
	if ev.Type == "deliverables/presented" {
		files := extractDeliverablePaths(ev.Data)
		if len(files) == 0 {
			return Message{}, false
		}
		return Message{
			Kind:      MessageKindDeliverables,
			Role:      "assistant",
			Seq:       ev.Seq,
			EventType: ev.Type,
			Raw:       append(json.RawMessage(nil), raw...),
			Body:      strings.Join(files, "\n"),
			Files:     files,
		}, true
	}
	if strings.HasPrefix(ev.Type, "compaction/") {
		body := ev.Type
		if t := extractTextish(ev.Data); t != "" {
			body = ev.Type + ": " + truncateRunes(t, 160)
		}
		return Message{
			Kind:      MessageKindCompaction,
			Role:      "system",
			Seq:       ev.Seq,
			EventType: ev.Type,
			Raw:       append(json.RawMessage(nil), raw...),
			Body:      body,
		}, true
	}
	role, body := summarizeEvent(ev.Type, ev.Data)
	switch ev.Type {
	case "assistant/thinking":
		if body == "" {
			return Message{}, false
		}
		return Message{
			Kind: MessageKindThinking,
			Role: "thinking",
			Seq:  ev.Seq,
			Body: body,
			Raw:  append(json.RawMessage(nil), raw...),
		}, true
	case "user/message", "user/prompt", "assistant/message":
		if body == "" {
			content := UserMessageContent(Message{Raw: raw})
			if ev.Type != "user/message" || !content.Structured || len(content.Attachments)+len(content.ExtraBlocks) == 0 {
				return Message{}, false
			}
		}
		msg := Message{
			Kind:      MessageKindText,
			Role:      role,
			Seq:       ev.Seq,
			EventType: ev.Type,
			Body:      body,
			Raw:       append(json.RawMessage(nil), raw...),
		}
		if ev.Type == "assistant/message" {
			var d struct {
				Message struct {
					Id string `json:"id"`
				} `json:"message"`
			}
			if json.Unmarshal(ev.Data, &d) == nil && d.Message.Id != "" {
				msg.ID = d.Message.Id
			}
		}
		return msg, true
	default:
		if body == "" {
			return Message{}, false
		}
		return Message{
			Kind: MessageKindText,
			Role: role,
			Seq:  ev.Seq,
			Body: body,
		}, true
	}
}

func extractDeliverablePaths(data json.RawMessage) []string {
	var d struct {
		Files []json.RawMessage `json:"files"`
	}
	if json.Unmarshal(data, &d) != nil {
		return nil
	}
	var out []string
	for _, raw := range d.Files {
		var f struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(raw, &f) == nil && strings.TrimSpace(f.Path) != "" {
			out = append(out, f.Path)
		}
	}
	return out
}

func isToolPersistenceType(typ string) bool {
	return strings.HasPrefix(typ, "tool/") || strings.HasPrefix(typ, "tool-workflow/")
}

func toolEventSummary(typ string, data json.RawMessage) string {
	switch typ {
	case "tool/call":
		var d struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}
		if json.Unmarshal(data, &d) == nil && d.Name != "" {
			return d.Name + " " + truncateRunes(d.Arguments, 80)
		}
	case "tool/result":
		if t := extractTextish(data); t != "" {
			return truncateRunes(t, 120)
		}
	default:
		if t := extractTextish(data); t != "" {
			return truncateRunes(t, 120)
		}
	}
	return typ
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}

func prettyJSONOrRaw(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	out := string(b)
	if len(out) > 4000 {
		return out[:4000] + "\n…"
	}
	return out
}
