package chat

import (
	"encoding/json"
	"math"
	"path"
	"sort"
	"strings"

	"whaleisle/internal/store/session"
)

type mentionEvent struct {
	sessionLevel bool
	Type         string          `json:"type"`
	Seq          float64         `json:"seq"`
	SurfaceOp    json.RawMessage `json:"surfaceOp"`
	Data         struct {
		Turn        *float64        `json:"turn"`
		CallID      string          `json:"callId"`
		Name        string          `json:"name"`
		Arguments   json.RawMessage `json:"arguments"`
		Meta        json.RawMessage `json:"meta"`
		IsError     bool            `json:"isError"`
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

// FileMentionResolver uses recorded turn locations and completed mutation facts.
// PTC inherits the wire location index, never a projection's TurnSeq, rootCallId,
// selected turn, or a path guessed from closing prose.
func FileMentionResolver(messages []session.Message, closing session.Message) func(string) string {
	var owner mentionEvent
	if closing.Role != "assistant" || closing.Streaming || json.Unmarshal(closing.Raw, &owner) != nil || owner.Type != "assistant/message" || owner.Data.Turn == nil || owner.Seq <= 0 || owner.Data.Interrupted {
		return nil
	}
	var events []mentionEvent
	for _, m := range messages {
		var e mentionEvent
		if json.Unmarshal(m.Raw, &e) == nil && e.Seq > 0 && e.Seq <= owner.Seq {
			var coordinates struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			_ = json.Unmarshal(m.Raw, &coordinates)
			e.sessionLevel = string(coordinates.Data["turn"]) == "null"
			events = append(events, e)
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Seq < events[j].Seq })
	var facts []mentionEvent
	var currentTurn *float64
	for _, e := range events {
		// ConversationLocationIndex updates its cursor from explicit Turn/Step
		// coordinates and retires it at the matching turn/end.
		if t := e.Data.Turn; t != nil && *t >= 0 && *t <= 9007199254740991 && math.Trunc(*t) == *t {
			currentTurn = t
		}
		turn := e.Data.Turn
		if e.Type == "tool/ptc-dispatch" && turn == nil && !e.sessionLevel {
			turn = currentTurn
		}
		if turn != nil && *turn == *owner.Data.Turn {
			facts = append(facts, e)
		}
		if e.Type == "turn/end" && currentTurn != nil && e.Data.Turn != nil && *currentTurn == *e.Data.Turn {
			currentTurn = nil
		}
	}
	paths := map[string]map[string]bool{}
	add := func(target, argument string) {
		if target == "" {
			return
		}
		for _, token := range []string{target, path.Base(strings.ReplaceAll(target, `\`, "/")), argument} {
			if token == "" {
				continue
			}
			if paths[token] == nil {
				paths[token] = map[string]bool{}
			}
			paths[token][target] = true
		}
	}
	type call struct {
		path string
		seq  float64
	}
	calls := map[string]call{}
	for _, e := range facts {
		if e.Type == "tool/call" && e.Data.CallID != "" {
			p := mutationMentionPath(e.Data.Name, rootMentionArguments(e.Data.Arguments))
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
					add(f.Path, "")
				}
			}
		case "tool/ptc-dispatch":
			if !e.Data.IsError {
				argument := mutationMentionPath(e.Data.Name, string(e.Data.Arguments))
				if argument != "" {
					add(mutationResultMentionPath(argument, e.Data.Meta), argument)
				}
			}
		case "tool/result":
			if string(e.SurfaceOp) != `"append"` || e.Data.Message.IsError || (len(e.Data.Error) > 0 && string(e.Data.Error) != "null") {
				continue
			}
			c := calls[e.Data.Message.Source.CallID]
			if c.path != "" && c.seq < e.Seq {
				add(mutationResultMentionPath(c.path, e.Data.Meta), c.path)
			}
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return func(value string) string {
		// Exact paths, basenames and argument aliases all participate in
		// ambiguity, as they do in producedFileMentions.
		if matches := paths[value]; len(matches) == 1 {
			for target := range matches {
				return target
			}
		}
		return ""
	}
}

func rootMentionArguments(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return ""
	}
	return text
}

func absoluteMentionPath(value string) bool {
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')
}

func mutationResultMentionPath(argument string, raw json.RawMessage) string {
	var meta map[string]json.RawMessage
	if json.Unmarshal(raw, &meta) != nil || meta == nil {
		return argument
	}
	value, exists := meta["path"]
	if !exists {
		return argument
	}
	var target string
	if json.Unmarshal(value, &target) == nil && absoluteMentionPath(target) {
		return target
	}
	if absoluteMentionPath(argument) {
		return argument
	}
	return ""
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
