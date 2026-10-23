package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf16"
	"whaleisle/internal/dsh"
)

// Inspector retains only the existing, authorized Session window. It never
// reads Host files, creates another subscription, or uploads log contents.
type sessionInspectorState struct {
	client        *dsh.Client
	session       string
	generation    uint64
	events        map[float64]json.RawMessage
	contextWaking map[float64]bool
	settledRoots  map[float64]string
}
type InspectorRecord struct {
	Key, Parent, Type  string
	Identity, Location string
	Seq, Time          float64
	Raw                json.RawMessage
	Disclosure         string
	CollapsedSummary   string
	ChatID, ChatPickID string
	ChatTarget         InspectorChatTarget
}
type InspectorSnapshot struct {
	Binding, SessionID string
	Records, Chat      []InspectorRecord
}

func (s *Store) resetInspectorLocked() {
	settled := s.inspector.settledRoots
	if s.inspector.client != s.client || s.inspector.session != s.selectedID || s.inspector.generation != s.followGeneration {
		settled = nil
	}
	s.inspector = sessionInspectorState{client: s.client, session: s.selectedID, generation: s.followGeneration, events: map[float64]json.RawMessage{}, settledRoots: settled}
}
func (s *Store) captureInspectorEventLocked(raw json.RawMessage) bool {
	var h struct {
		Type string
		Seq  *float64
	}
	if json.Unmarshal(raw, &h) != nil || h.Type == "" || h.Seq == nil || *h.Seq < 0 {
		return false
	}
	seq := *h.Seq
	if s.inspector.events == nil || s.inspector.client != s.client || s.inspector.session != s.selectedID || s.inspector.generation != s.followGeneration {
		s.resetInspectorLocked()
	}
	if string(s.inspector.events[seq]) == string(raw) {
		return false
	}
	// Follow/page hand us immutable decoded event bytes. Retain that window
	// without duplicating its payload; snapshots copy before exposing it.
	s.inspector.events[seq] = raw
	s.inspector.contextWaking = nil
	return true
}
func inspectorRecord(key, parent string, raw json.RawMessage) InspectorRecord {
	var h struct {
		Type string
		Seq  *float64
		Time float64
		Data struct{ Turn, Step json.RawMessage }
	}
	_ = json.Unmarshal(raw, &h)
	row := InspectorRecord{Key: key, Parent: parent, Type: h.Type, Time: h.Time, Raw: append(json.RawMessage(nil), raw...), ChatTarget: inspectorEventChatTarget(raw)}
	if h.Seq != nil {
		row.Seq = *h.Seq
		row.Identity = fmt.Sprintf("%.0f", *h.Seq)
	}
	if len(h.Data.Turn) > 0 {
		row.Location = string(h.Data.Turn)
		if len(h.Data.Step) > 0 {
			row.Location += "/" + string(h.Data.Step)
		}
	}
	return row
}
func inspectorChildren(root InspectorRecord) []InspectorRecord {
	var ev struct {
		Data struct{ Stream []json.RawMessage }
	}
	_ = json.Unmarshal(root.Raw, &ev)
	rows := []InspectorRecord{root}
	type block struct {
		key, reasoning string
		isReasoning    bool
		position       int
		children       []InspectorRecord
	}
	blocks := map[int]*block{}
	byKey := map[string]*block{}
	chunks, valid := expandAssistantStream(ev.Data.Stream)
	if !valid {
		return rows
	}
	for i, item := range chunks {
		row := inspectorRecord(fmt.Sprintf("%s/chunk:%d", root.Key, i), root.Key, item.chunk)
		row.Seq, row.Identity, row.Location, row.Time = root.Seq, fmt.Sprint(i), root.Location, item.time
		row.ChatTarget = inspectorChunkChatTarget(root.ChatTarget, item.chunk)
		var h struct {
			Type, BlockType, Text string
			Index                 *int
		}
		_ = json.Unmarshal(item.chunk, &h)
		var owner *block
		if h.Type == "block-start" && h.Index != nil {
			owner = &block{key: row.Key, isReasoning: h.BlockType == "reasoning", position: len(rows)}
			blocks[*h.Index], byKey[row.Key] = owner, owner
			row.Disclosure = "open"
			rows = append(rows, row)
		} else {
			if h.Index != nil {
				owner = blocks[*h.Index]
			}
			if owner == nil {
				rows = append(rows, row)
			} else {
				row.Parent = owner.key
				owner.children = append(owner.children, row)
				if h.Type == "block-end" {
					rows[owner.position].Disclosure = "closed"
				}
			}
		}
		if owner != nil && owner.isReasoning && h.Type == "reasoning-delta" {
			owner.reasoning = inspectorStreamPrefix(owner.reasoning+h.Text, 240)
		}
	}
	out := make([]InspectorRecord, 0, len(chunks)+1)
	for _, row := range rows {
		if b := byKey[row.Key]; b != nil {
			if b.isReasoning {
				row.CollapsedSummary = "blockType: reasoning"
				if b.reasoning != "" {
					row.CollapsedSummary = inspectorStreamPrefix(row.CollapsedSummary+" · "+b.reasoning, 240)
				}
			}
			out = append(out, row)
			out = append(out, b.children...)
		} else {
			out = append(out, row)
		}
	}
	return out
}

func inspectorStreamPrefix(text string, limit int) string {
	units := utf16.Encode([]rune(text))
	if len(units) <= limit {
		return text
	}
	return string(utf16.Decode(units[:limit]))
}

// Settlement is named by the followed stream protocol, never inferred from text.
func (s *Store) settleInspectorAttemptLocked(attempt *assistantStreamAttempt, seq float64) {
	if attempt == nil || len(attempt.Stream) == 0 || s.inspector.client != s.client || s.inspector.session != s.selectedID || s.inspector.generation != s.followGeneration {
		return
	}
	if s.inspector.settledRoots == nil {
		s.inspector.settledRoots = make(map[float64]string)
	}
	s.inspector.settledRoots[seq] = "attempt:" + attempt.AttemptID
}
func (s *Store) pruneInspectorSettlementsLocked() {
	for seq := range s.inspector.settledRoots {
		if _, retained := s.inspector.events[seq]; !retained {
			delete(s.inspector.settledRoots, seq)
		}
	}
}

func (s *Store) inspectorCursorLocked() *float64 {
	cursor := float64(-1)
	for seq := range s.inspector.events {
		cursor = max(cursor, seq)
	}
	return &cursor
}

func assistantBlockVisible(block assistantStreamBlock) bool {
	return block.text != "" || block.imageRef != "" || len(block.raw) > 0
}

func inspectorLiveRecords(a *assistantStreamAttempt, parent string) []InspectorRecord {
	chunks, valid := expandAssistantStream(a.Stream)
	if !valid || len(chunks) == 0 {
		return nil
	}
	raw, _ := json.Marshal(map[string]any{"data": map[string]any{"stream": a.Stream}})
	root := InspectorRecord{Key: "attempt:" + a.AttemptID, Parent: parent, Type: "assistant/live", Identity: a.AttemptID, Location: fmt.Sprintf("%.0f/%.0f", a.Turn, a.Step), Time: chunks[len(chunks)-1].time, Raw: raw, ChatTarget: InspectorChatTarget{NodeKind: "assistant-step", Turn: &a.Turn, Step: &a.Step, LiveAssistant: true}}
	rows := inspectorChildren(root)
	for i := range rows {
		var h struct{ Index *int }
		_ = json.Unmarshal(rows[i].Raw, &h)
		if h.Index != nil && assistantBlockVisible(a.blocks[*h.Index]) {
			rows[i].ChatID, rows[i].ChatPickID = a.rows[*h.Index], a.rows[*h.Index]
		}
	}
	for _, item := range chunks {
		var h struct{ Index *int }
		_ = json.Unmarshal(item.chunk, &h)
		if h.Index != nil && a.rows[*h.Index] != "" && assistantBlockVisible(a.blocks[*h.Index]) {
			rows[0].ChatID = a.rows[*h.Index]
			break
		}
	}
	for i := range rows {
		if rows[i].ChatID == "" {
			rows[i].ChatID = rows[0].ChatID
		}
		if a.settlementSeq > 0 {
			// Successful settlement replaces native prefix Message IDs immediately.
			// The retained raw live rows reveal that exact canonical event until
			// Step end removes them from the log.
			rows[i].ChatID, rows[i].ChatPickID = "", ""
			rows[i].Seq = a.settlementSeq
			rows[i].ChatTarget.Seq, rows[i].ChatTarget.AnchorSeq = a.settlementSeq, &a.settlementSeq
		}
	}
	rows[0].Raw, _ = json.Marshal(struct {
		AttemptID string  `json:"attemptId"`
		Turn      float64 `json:"turn"`
		Step      float64 `json:"step"`
		Chunks    int     `json:"chunks"`
	}{a.AttemptID, a.Turn, a.Step, len(chunks)})
	return rows
}

func (s *Store) InspectorSnapshot() InspectorSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := InspectorSnapshot{Binding: fmt.Sprintf("%p/%s/%d", s.client, s.selectedID, s.followGeneration), SessionID: s.selectedID}
	if s.inspector.client == s.client && s.inspector.session == s.selectedID && s.inspector.generation == s.followGeneration {
		s.pruneInspectorSettlementsLocked()
		seqs := make([]float64, 0, len(s.inspector.events))
		for seq := range s.inspector.events {
			seqs = append(seqs, seq)
		}
		sort.Float64s(seqs)
		turn, step := "", ""
		active := s.assistantStream.active
		if active == nil {
			active = s.assistantStream.retained
		}
		addLive := func() {
			if active != nil {
				parent := step
				if parent == "" {
					parent = turn
				}
				out.Records = append(out.Records, inspectorLiveRecords(active, parent)...)
				active = nil
			}
		}
		for _, seq := range seqs {
			if active != nil && active.inspectorAfterSeq != nil && seq > *active.inspectorAfterSeq {
				addLive()
			}
			key := s.inspector.settledRoots[seq]
			if key == "" {
				key = fmt.Sprintf("event:%.0f", seq)
			}
			row := inspectorRecord(key, "", s.inspector.events[seq])
			if row.Type == "turn/start" {
				row.Disclosure = "open"
				turn = row.Key
				step = ""
			} else if row.Type == "step/start" {
				row.Disclosure = "open"
				row.Parent = turn
				step = row.Key
			} else if row.Type == "turn/end" {
				row.Parent = turn
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
		addLive()
	} else if a := s.assistantStream.active; a != nil {
		out.Records = append(out.Records, inspectorLiveRecords(a, "")...)
	}
	// Inspect actual native Chat projection objects, without inventing web Nodes.
	for _, msg := range projectCommandMessages(s.projectAssistantMessagesLocked(projectMessageEdits(s.messages))) {
		raw, _ := json.Marshal(msg)
		row := inspectorRecord("chat:"+msg.ID, "", raw)
		row.Type = string(msg.Kind)
		row.Seq = msg.Seq
		row.ChatTarget = msg.InspectorTarget()
		out.Chat = append(out.Chat, row)
	}
	return out
}

func (s *Store) InspectorBinding() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("%p/%s/%d", s.client, s.selectedID, s.followGeneration)
}
