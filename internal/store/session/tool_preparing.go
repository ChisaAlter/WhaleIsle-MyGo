package session

import (
	"encoding/json"
	"strings"
)

// This state belongs to one already-validated assistant attempt. Revision,
// nextIndex, baseline replay and attempt disposal remain assistant_stream's job.
type toolPreparingState struct {
	calls map[string]toolPreparingCall
	order []string
}

type toolPreparingCall struct {
	name, args string
	time       float64
	sealed     bool
}

// foldToolPreparingChunk follows the provider-neutral StreamChunk contract:
// tool-call-delta.id and block-end.block.id are real call IDs, not text matches.
// Like the original Tool node, unnamed prefixes do not open a preparing row.
func foldToolPreparingChunk(state *toolPreparingState, raw json.RawMessage, time float64) bool {
	var chunk struct {
		Type  string  `json:"type"`
		Index *int    `json:"index"`
		ID    string  `json:"id"`
		Name  *string `json:"name"`
		Delta *string `json:"argumentsDelta"`
		Block struct {
			Type      string  `json:"type"`
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			Arguments *string `json:"arguments"`
		} `json:"block"`
	}
	if state == nil || json.Unmarshal(raw, &chunk) != nil || chunk.Index == nil || *chunk.Index < 0 {
		return false
	}
	switch chunk.Type {
	case "tool-call-delta":
		if strings.TrimSpace(chunk.ID) == "" || chunk.Delta == nil {
			return false
		}
		call, exists := state.calls[chunk.ID]
		if !exists {
			if chunk.Name == nil || strings.TrimSpace(*chunk.Name) == "" {
				return false
			}
			call = toolPreparingCall{name: *chunk.Name, time: time}
			if state.calls == nil {
				state.calls = make(map[string]toolPreparingCall)
			}
			state.order = append(state.order, chunk.ID)
		}
		if call.sealed {
			return false
		}
		call.args += *chunk.Delta
		state.calls[chunk.ID] = call
		return true
	case "block-end":
		block := chunk.Block
		if block.Type != "tool-call" || strings.TrimSpace(block.ID) == "" || strings.TrimSpace(block.Name) == "" || block.Arguments == nil {
			return false
		}
		call, exists := state.calls[block.ID]
		if !exists || call.sealed {
			return false
		}
		call.name, call.args, call.sealed = block.Name, *block.Arguments, true
		state.calls[block.ID] = call
		return true
	default:
		return false
	}
}

// appendToolPreparingMessages does not manufacture persistence events. Tool
// metadata is a disposable UI projection; the journal and paging Seq stay intact.
func appendToolPreparingMessages(messages []Message, state toolPreparingState, turnSeq float64) []Message {
	if len(state.order) == 0 {
		return messages
	}
	owned := make(map[string]bool)
	for _, message := range messages {
		if message.Tool != nil {
			owned[message.Tool.CallID] = true
		}
		if part, ok := readToolLifecyclePart(message); ok {
			owned[part.id] = true
		}
	}
	out := append([]Message(nil), messages...)
	for _, id := range state.order {
		if owned[id] {
			continue
		}
		call := state.calls[id]
		view := &ToolCallView{CallID: id, Name: call.name, Arguments: call.args, State: "preparing"}
		out = append(out, Message{ID: "tool:" + id, Kind: MessageKindTool, Role: "tool", Body: call.name, Args: call.args, Tool: view, Streaming: true, TurnSeq: turnSeq, EventTime: call.time})
	}
	return out
}

// projectPreparingToolsLocked reads the same active/retained ownership that
// already controls answer settlement. It does not add rows to either ownership
// map, so tool chunks can never become a text/reasoning settlement anchor.
func (s *Store) projectPreparingToolsLocked(messages []Message) []Message {
	if s.assistantStream.rebaseline {
		return messages
	}
	attempt := s.assistantStream.active
	if attempt == nil {
		attempt = s.assistantStream.retained
	}
	if attempt == nil || toolPreparingAttemptClosed(messages, attempt) {
		return messages
	}
	return appendToolPreparingMessages(messages, attempt.tools, s.editTurnSeq)
}

func toolPreparingAttemptClosed(messages []Message, attempt *assistantStreamAttempt) bool {
	for _, message := range messages {
		var event struct {
			Type string  `json:"type"`
			Seq  float64 `json:"seq"`
			Data struct {
				Turn float64 `json:"turn"`
				Step float64 `json:"step"`
			} `json:"data"`
		}
		if json.Unmarshal(message.Raw, &event) != nil || event.Seq <= attempt.StartedAfterSeq || event.Data.Turn != attempt.Turn {
			continue
		}
		if event.Type == "turn/end" || (event.Type == "step/end" && event.Data.Step == attempt.Step) {
			return true
		}
	}
	return false
}
