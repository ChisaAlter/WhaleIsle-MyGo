package session

import (
	"encoding/json"
	"strings"
)

// MessageKindCommand is a durable human-command lifecycle, not a model turn.
const MessageKindCommand MessageKind = "command"

// CommandView pairs command/run and command/done by the Host-owned commandId.
// SourceEventSeq retains a successful command's richer domain-event reference.
type CommandView struct {
	CommandID      string
	Name           string
	Args           string
	State          string // running, success, error
	Text           string
	SourceEventSeq *float64
}

func classifyCommandEvent(raw json.RawMessage) (Message, bool) {
	var event struct {
		Type string  `json:"type"`
		Seq  float64 `json:"seq"`
		Time float64 `json:"time"`
		Data struct {
			CommandID      string   `json:"commandId"`
			Name           string   `json:"name"`
			Args           string   `json:"args"`
			Kind           string   `json:"kind"`
			Text           string   `json:"text"`
			SourceEventSeq *float64 `json:"sourceEventSeq"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &event) != nil || strings.TrimSpace(event.Data.CommandID) == "" {
		return Message{}, false
	}
	command := CommandView{CommandID: event.Data.CommandID}
	switch event.Type {
	case "command/run":
		if strings.TrimSpace(event.Data.Name) == "" {
			return Message{}, false
		}
		command.Name, command.Args, command.State = event.Data.Name, event.Data.Args, "running"
	case "command/done":
		if event.Data.Kind != "success" && event.Data.Kind != "error" {
			return Message{}, false
		}
		command.State, command.Text = event.Data.Kind, event.Data.Text
		if command.State == "success" && event.Data.SourceEventSeq != nil && *event.Data.SourceEventSeq >= 0 {
			seq := *event.Data.SourceEventSeq
			command.SourceEventSeq = &seq
		}
	default:
		return Message{}, false
	}
	return Message{
		Kind: MessageKindCommand, Role: "command", Seq: event.Seq,
		EventType: event.Type, EventTime: event.Time,
		Raw: append(json.RawMessage(nil), raw...), Command: &command,
	}, true
}

// projectCommandMessages leaves the append-only journal and its paging cursor
// untouched. A done-only history page remains visible; loading the older run
// enriches the same identity rather than creating a duplicate result row.
func projectCommandMessages(messages []Message) []Message {
	type pair struct {
		run, done int
	}
	pairs := make(map[string]pair)
	for i, msg := range messages {
		if msg.Kind != MessageKindCommand || msg.Command == nil {
			continue
		}
		id := msg.Command.CommandID
		p, ok := pairs[id]
		if !ok {
			p = pair{run: -1, done: -1}
		}
		if msg.EventType == "command/run" && (p.run < 0 || msg.Seq < messages[p.run].Seq) {
			p.run = i
		}
		if msg.EventType == "command/done" && (p.done < 0 || msg.Seq >= messages[p.done].Seq) {
			p.done = i
		}
		pairs[id] = p
	}
	out := make([]Message, 0, len(messages))
	for i, msg := range messages {
		if msg.Kind != MessageKindCommand || msg.Command == nil {
			out = append(out, msg)
			continue
		}
		p := pairs[msg.Command.CommandID]
		anchor := p.run
		if anchor < 0 {
			anchor = p.done
		}
		if i != anchor {
			continue
		}
		view := *msg.Command
		if p.done >= 0 {
			done := messages[p.done].Command
			view.State, view.Text = done.State, done.Text
			view.SourceEventSeq = done.SourceEventSeq
		}
		if view.SourceEventSeq != nil {
			seq := *view.SourceEventSeq
			view.SourceEventSeq = &seq
		}
		msg.Command = &view
		msg.ID = "command:" + view.CommandID
		msg.Body = view.Text
		if msg.Body == "" && view.Name != "" {
			msg.Body = "/" + view.Name
		}
		out = append(out, msg)
	}
	return out
}
