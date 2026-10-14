package chat

import (
	"encoding/json"
	"path"
	"strings"

	"whaleisle/internal/store/session"
)

type mentionEvent struct {
	Type      string          `json:"type"`
	Seq       float64         `json:"seq"`
	SurfaceOp json.RawMessage `json:"surfaceOp"`
	Data      struct {
		Turn        *float64        `json:"turn"`
		CallID      string          `json:"callId"`
		Name        string          `json:"name"`
		Arguments   string          `json:"arguments"`
		Error       json.RawMessage `json:"error"`
		Interrupted bool            `json:"interrupted"`
		Files       []struct {
			Path string `json:"path"`
		} `json:"files"`
		Message struct {
			IsError bool `json:"isError"`
			Source  struct {
				CallID string `json:"callId"`
			} `json:"source"`
		} `json:"message"`
	} `json:"data"`
}

// FileMentionResolver uses explicit wire turn ownership and completed facts only.
// Missing turn metadata (including unresolved PTC children) never borrows a
// neighboring message's TurnSeq, rootCallId, or the currently selected turn.
func FileMentionResolver(messages []session.Message, closing session.Message) func(string) string {
	var owner mentionEvent
	if closing.Role != "assistant" || closing.Streaming || json.Unmarshal(closing.Raw, &owner) != nil || owner.Type != "assistant/message" || owner.Data.Turn == nil || owner.Seq <= 0 || owner.Data.Interrupted {
		return nil
	}
	var facts []mentionEvent
	for _, m := range messages {
		var e mentionEvent
		if json.Unmarshal(m.Raw, &e) == nil && e.Data.Turn != nil && *e.Data.Turn == *owner.Data.Turn && e.Seq > 0 && e.Seq <= owner.Seq {
			facts = append(facts, e)
		}
	}
	paths := map[string]bool{}
	type call struct {
		path string
		seq  float64
	}
	calls := map[string]call{}
	for _, e := range facts {
		if e.Type == "tool/call" && e.Data.CallID != "" {
			p := mutationMentionPath(e.Data.Name, e.Data.Arguments)
			// Duplicate call IDs are not enough authority to choose one file.
			if _, exists := calls[e.Data.CallID]; exists {
				p = ""
			}
			calls[e.Data.CallID] = call{p, e.Seq}
		}
	}
	for _, e := range facts {
		switch e.Type {
		case "deliverables/presented":
			for _, f := range e.Data.Files {
				if strings.TrimSpace(f.Path) != "" {
					paths[f.Path] = true
				}
			}
		case "tool/result":
			if string(e.SurfaceOp) != `"append"` || e.Data.Message.IsError || (len(e.Data.Error) > 0 && string(e.Data.Error) != "null") {
				continue
			}
			c := calls[e.Data.Message.Source.CallID]
			if c.path != "" && c.seq < e.Seq {
				paths[c.path] = true
			}
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return func(value string) string {
		if paths[value] {
			return value
		}
		match := ""
		for p := range paths {
			if path.Base(strings.ReplaceAll(p, `\`, "/")) == value {
				if match != "" {
					return ""
				}
				match = p
			}
		}
		return match
	}
}

func mutationMentionPath(name, raw string) string {
	var a map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &a) != nil || a == nil {
		return ""
	}
	str := func(key string) (string, bool) {
		var s string
		v, ok := a[key]
		if !ok || string(v) == "null" || json.Unmarshal(v, &s) != nil {
			return "", false
		}
		return s, true
	}
	valid := false
	key := "file_path"
	switch name {
	case "write":
		_, valid = str("content")
	case "edit":
		old, ok := str("old_string")
		next, nok := str("new_string")
		valid = ok && nok && old != "" && old != next
		if v, exists := a["replace_all"]; exists {
			var b bool
			valid = valid && string(v) != "null" && json.Unmarshal(v, &b) == nil
		}
	case "str_replace_editor":
		key = "path"
		command, _ := str("command")
		switch command {
		case "create":
			_, valid = str("file_text")
		case "str_replace":
			old, ok := str("old_str")
			_, nok := str("new_str")
			_, exists := a["new_str"]
			valid = ok && old != "" && (!exists || nok)
		case "insert":
			var line int
			v, exists := a["insert_line"]
			_, ok := str("new_str")
			valid = exists && string(v) != "null" && json.Unmarshal(v, &line) == nil && line >= 0 && ok
		}
	}
	p, ok := str(key)
	if !valid || !ok || strings.TrimSpace(p) == "" {
		return ""
	}
	return p
}
