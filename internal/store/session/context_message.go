package session

import (
	"encoding/json"
	"sort"
	"strings"
)

// Context is logged model-facing input from a producer, never human speech.
const MessageKindContext MessageKind = "context"

type ContextField struct{ Name, Value string }
type ContextMessageView struct {
	Producer, Role, Form, Body string
	Fields                     []ContextField
}

type contextEnvelope struct {
	Type string  `json:"type"`
	Seq  float64 `json:"seq"`
	Time float64 `json:"time"`
	Data struct {
		ID      string            `json:"id"`
		Source  json.RawMessage   `json:"source"`
		Content []json.RawMessage `json:"content"`
		Message *struct {
			ID      string            `json:"id"`
			Source  json.RawMessage   `json:"source"`
			Content []json.RawMessage `json:"content"`
		} `json:"message"`
	} `json:"data"`
}

func readContextEnvelope(raw json.RawMessage) (contextEnvelope, bool) {
	var e contextEnvelope
	if json.Unmarshal(raw, &e) != nil {
		return e, false
	}
	if e.Type == "developer/message" {
		if e.Data.Message == nil {
			return e, false
		}
		e.Data.ID, e.Data.Source, e.Data.Content = e.Data.Message.ID, e.Data.Message.Source, e.Data.Message.Content
		return e, true
	}
	if e.Type != "user/message" || len(e.Data.Source) == 0 {
		return e, false
	} // legacy human logs
	var source struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(e.Data.Source, &source)
	return e, source.Kind != "user"
}

func classifyContextMessage(raw json.RawMessage) (Message, bool) {
	e, ok := readContextEnvelope(raw)
	if !ok {
		return Message{}, false
	}
	return Message{Kind: MessageKindContext, Role: "context", Seq: e.Seq, EventTime: e.Time, EventType: e.Type, SourceMessageID: e.Data.ID, Raw: append(json.RawMessage(nil), raw...), Body: contextContent(e.Data.Content)}, true
}

func contextContent(content []json.RawMessage) string {
	var parts []string
	for _, raw := range content {
		var part struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(raw, &part) == nil && part.Type == "text" {
			parts = append(parts, part.Text)
		} else {
			parts = append(parts, prettyJSONOrRaw(string(raw)))
		}
	}
	return strings.Join(parts, "\n")
}

// ContextMessageDetails derives display-only metadata from the retained event.
// Unknown or malformed producer forms remain readable as opaque source fields.
// No metadata can create a file action, a navigation target, or a user edit.
func ContextMessageDetails(msg Message) ContextMessageView {
	v := ContextMessageView{Role: "inject", Body: msg.Body}
	e, ok := readContextEnvelope(msg.Raw)
	if !ok {
		v.Producer = "unknown"
		return v
	}
	var source map[string]json.RawMessage
	if json.Unmarshal(e.Data.Source, &source) != nil || source == nil {
		v.Producer = "unknown"
		if len(e.Data.Source) > 0 {
			v.Fields = append(v.Fields, ContextField{"source", string(e.Data.Source)})
		}
		return v
	}
	str := func(key string) string { var s string; _ = json.Unmarshal(source[key], &s); return s }
	kind := str("kind")
	v.Producer = kind
	if v.Producer == "" {
		v.Producer = "unknown"
	}
	collect := func(member, key string) string {
		var entries []map[string]json.RawMessage
		if json.Unmarshal(source[member], &entries) != nil {
			return ""
		}
		seen := map[string]bool{}
		var labels []string
		for _, entry := range entries {
			var value string
			_ = json.Unmarshal(entry[key], &value)
			if value != "" && !seen[value] {
				seen[value] = true
				labels = append(labels, value)
			}
		}
		return strings.Join(labels, ", ")
	}
	var label string
	switch kind {
	case "session-reference":
		v.Role = "recall"
		label = collect("references", "label")
	case "agent-instructions":
		label = collect("changes", "path")
	case "skill-invocation":
		label = str("name")
	}
	if label != "" {
		v.Producer = label
	}
	switch form := str("form"); form {
	case "instructions", "catalog", "snapshot", "notice", "relay", "recall":
		v.Form = form
	}
	// Retain kind/form and all unknown fields as evidence. Structured producer
	// arrays stay intact, including recall omitted counts and snapshot sections.
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		v.Fields = append(v.Fields, ContextField{key, prettyJSONOrRaw(string(source[key]))})
	}
	return v
}
