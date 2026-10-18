package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"whaleisle/internal/dsh"
)

// Inspector retains only the existing, authorized Session window. It never
// reads Host files, creates another subscription, or uploads log contents.
type sessionInspectorState struct {
	client     *dsh.Client
	session    string
	generation uint64
	events     map[float64]json.RawMessage
}
type InspectorRecord struct {
	Key, Parent, Type string
	Seq, Time         float64
	Raw               json.RawMessage
}
type InspectorSnapshot struct {
	Binding, SessionID string
	Records, Chat      []InspectorRecord
}

func (s *Store) resetInspectorLocked() {
	s.inspector = sessionInspectorState{client: s.client, session: s.selectedID, generation: s.followGeneration, events: map[float64]json.RawMessage{}}
}
func (s *Store) captureInspectorEventLocked(raw json.RawMessage) bool {
	var h struct {
		Type string
		Seq  float64
	}
	if json.Unmarshal(raw, &h) != nil || h.Type == "" || h.Seq <= 0 {
		return false
	}
	if s.inspector.events == nil || s.inspector.client != s.client || s.inspector.session != s.selectedID || s.inspector.generation != s.followGeneration {
		s.resetInspectorLocked()
	}
	if string(s.inspector.events[h.Seq]) == string(raw) {
		return false
	}
	// Follow/page hand us immutable decoded event bytes. Retain that window
	// without duplicating its payload; snapshots copy before exposing it.
	s.inspector.events[h.Seq] = raw
	return true
}
func inspectorRecord(key, parent string, raw json.RawMessage) InspectorRecord {
	var h struct {
		Type      string
		Seq, Time float64
	}
	_ = json.Unmarshal(raw, &h)
	return InspectorRecord{Key: key, Parent: parent, Type: h.Type, Seq: h.Seq, Time: h.Time, Raw: append(json.RawMessage(nil), raw...)}
}
func inspectorChildren(root InspectorRecord) []InspectorRecord {
	var ev struct {
		Data struct{ Stream []json.RawMessage }
	}
	_ = json.Unmarshal(root.Raw, &ev)
	rows := []InspectorRecord{root}
	blocks := map[int]string{}
	chunks, valid := expandAssistantStream(ev.Data.Stream)
	if !valid {
		return rows
	}
	for i, item := range chunks {
		chunk, timestamp := item.chunk, item.time
		row := inspectorRecord(fmt.Sprintf("%s/chunk:%d", root.Key, i), root.Key, chunk)
		row.Seq = root.Seq
		row.Time = timestamp
		var h struct {
			Type  string
			Index int
		}
		_ = json.Unmarshal(chunk, &h)
		if h.Type == "block-start" {
			blocks[h.Index] = row.Key
		} else if key := blocks[h.Index]; key != "" {
			row.Parent = key
		}
		rows = append(rows, row)
	}
	return rows
}
func (s *Store) InspectorSnapshot() InspectorSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := InspectorSnapshot{Binding: fmt.Sprintf("%p/%s/%d", s.client, s.selectedID, s.followGeneration), SessionID: s.selectedID}
	if s.inspector.client == s.client && s.inspector.session == s.selectedID && s.inspector.generation == s.followGeneration {
		seqs := make([]float64, 0, len(s.inspector.events))
		for seq := range s.inspector.events {
			seqs = append(seqs, seq)
		}
		sort.Float64s(seqs)
		turn, step := "", ""
		for _, seq := range seqs {
			row := inspectorRecord(fmt.Sprintf("event:%.0f", seq), "", s.inspector.events[seq])
			if row.Type == "turn/start" {
				turn = row.Key
				step = ""
			} else if row.Type == "step/start" {
				row.Parent = turn
				step = row.Key
			} else if step != "" {
				row.Parent = step
			} else {
				row.Parent = turn
			}
			out.Records = append(out.Records, inspectorChildren(row)...)
			if row.Type == "step/end" {
				step = ""
			}
			if row.Type == "turn/end" {
				turn = ""
				step = ""
			}
		}
	}
	if a := s.assistantStream.active; a != nil {
		raw, _ := json.Marshal(map[string]any{"type": "assistant/stream", "data": a})
		out.Records = append(out.Records, inspectorChildren(inspectorRecord("attempt:"+a.AttemptID, "", raw))...)
	}
	// Inspect actual native Chat projection objects, without inventing web Nodes.
	for _, msg := range projectCommandMessages(s.projectAssistantMessagesLocked(projectMessageEdits(s.messages))) {
		raw, _ := json.Marshal(msg)
		row := inspectorRecord("chat:"+msg.ID, "", raw)
		row.Type = string(msg.Kind)
		row.Seq = msg.Seq
		out.Chat = append(out.Chat, row)
	}
	return out
}

func (s *Store) InspectorBinding() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("%p/%s/%d", s.client, s.selectedID, s.followGeneration)
}
