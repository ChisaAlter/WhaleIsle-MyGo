package session

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TrajectoryTiming contains recorded Host timestamps, never local receipt time,
// kind weights, adjacent-event estimates or a running wall-clock extrapolation.
// Nil boundaries are unknown, including truncated historical windows.
type TrajectoryTiming struct {
	OperationID                          string
	StartedAt, CompletedAt, FirstTokenAt *float64
	Closed, Invalid                      bool
}

func (t *TrajectoryTiming) DurationMillis() (float64, bool) {
	if t == nil || t.Invalid || t.StartedAt == nil || t.CompletedAt == nil || *t.CompletedAt < *t.StartedAt {
		return 0, false
	}
	return *t.CompletedAt - *t.StartedAt, true
}

func trajectoryTimestamp(value *float64) *float64 {
	if value == nil || !assistantSafeInt(*value) || *value < 0 {
		return nil
	}
	copy := *value
	return &copy
}

type trajectoryTimeEvent struct {
	Type string   `json:"type"`
	Seq  float64  `json:"seq"`
	Time *float64 `json:"time"`
	Data struct {
		Turn, Step   *float64
		CompactionID string            `json:"compactionId"`
		Stream       []json.RawMessage `json:"stream"`
	} `json:"data"`
}

func trajectoryStepID(turn, step *float64) string {
	if turn == nil || step == nil || !assistantSafeInt(*turn) || !assistantSafeInt(*step) || *turn < 0 || *step < 0 {
		return ""
	}
	return fmt.Sprintf("assistant:%.0f:%.0f", *turn, *step)
}

func trajectoryEventID(e trajectoryTimeEvent, raw json.RawMessage) string {
	switch e.Type {
	case "step/start", "step/end", "assistant/message", "assistant/attempt":
		return trajectoryStepID(e.Data.Turn, e.Data.Step)
	case "tool/call", "tool/result", "tool/ptc-dispatch-start", "tool/ptc-dispatch":
		part, ok := readToolLifecyclePart(Message{Raw: raw})
		if ok {
			return "tool:" + part.id
		}
	case "user/message":
		var d struct {
			Data struct {
				Source struct {
					Kind         string `json:"kind"`
					CompactionID string `json:"compactionId"`
				} `json:"source"`
			} `json:"data"`
		}
		if json.Unmarshal(raw, &d) == nil && d.Data.Source.Kind == "compact-checkpoint" && d.Data.Source.CompactionID != "" {
			return "compaction:" + d.Data.Source.CompactionID
		}
	case "compaction/start", "compaction/summary", "compaction/end":
		if e.Data.CompactionID != "" {
			return "compaction:" + e.Data.CompactionID
		}
	}
	return ""
}

// These boundaries are not chat messages. Retain them separately so follow and
// session/page can hydrate timing without changing chat/paging/edit identities.
func (s *Store) captureTrajectoryBoundaryLocked(raw json.RawMessage) bool {
	var e trajectoryTimeEvent
	if json.Unmarshal(raw, &e) != nil || e.Seq <= 0 {
		return false
	}
	switch e.Type {
	case "turn/start", "turn/end", "step/start", "step/end", "assistant/message", "assistant/attempt", "llm/retry", "llm/retry-started", "compaction/start", "compaction/summary", "compaction/end", "session/end":
	default:
		var ok bool
		raw, ok = sanitizedPromptBoundary(raw)
		if !ok {
			return false
		}
	}
	if s.trajectoryBoundaries == nil {
		s.trajectoryBoundaries = make(map[float64]json.RawMessage)
	}
	if string(s.trajectoryBoundaries[e.Seq]) == string(raw) {
		return false
	}
	s.trajectoryBoundaries[e.Seq] = append(json.RawMessage(nil), raw...)
	return true
}

func trajectoryTokenTime(raw json.RawMessage, timestamp float64) *float64 {
	var c struct {
		Type           string  `json:"type"`
		Text           string  `json:"text"`
		ArgumentsDelta string  `json:"argumentsDelta"`
		Name           *string `json:"name"`
	}
	if json.Unmarshal(raw, &c) != nil {
		return nil
	}
	token := (c.Type == "text-delta" || c.Type == "reasoning-delta") && c.Text != ""
	token = token || c.Type == "tool-call-delta" && (c.ArgumentsDelta != "" || c.Name != nil)
	if !token {
		return nil
	}
	return trajectoryTimestamp(&timestamp)
}

func trajectoryStreamFirstToken(stream []json.RawMessage) *float64 {
	chunks, ok := expandAssistantStream(stream)
	if !ok {
		return nil
	}
	for _, c := range chunks {
		if first := trajectoryTokenTime(c.chunk, c.time); first != nil {
			return first
		}
	}
	return nil
}

// Copy scalars so the snapshot can be read outside the Store mutex.
func cloneTrajectoryTiming(t *TrajectoryTiming) *TrajectoryTiming {
	if t == nil {
		return nil
	}
	copy := *t
	copy.StartedAt = trajectoryTimestamp(t.StartedAt)
	copy.CompletedAt = trajectoryTimestamp(t.CompletedAt)
	copy.FirstTokenAt = trajectoryTimestamp(t.FirstTokenAt)
	return &copy
}

func (s *Store) projectTrajectoryTimingLocked(messages []Message) []Message {
	raws := make(map[float64]json.RawMessage, len(s.trajectoryBoundaries)+len(s.messages))
	for seq, raw := range s.trajectoryBoundaries {
		raws[seq] = raw
	}
	for _, m := range s.messages {
		if m.Seq > 0 && len(m.Raw) > 0 {
			raws[m.Seq] = m.Raw
		}
	}
	seqs := make([]float64, 0, len(raws))
	for seq := range raws {
		seqs = append(seqs, seq)
	}
	sort.Float64s(seqs)
	ops := make(map[string]*TrajectoryTiming)
	anchors := make(map[string]json.RawMessage)
	starts := make(map[string]bool)
	finals := make(map[string]bool)
	for _, seq := range seqs {
		raw := raws[seq]
		var e trajectoryTimeEvent
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		id := trajectoryEventID(e, raw)
		if id == "" {
			continue
		}
		if strings.HasPrefix(id, "compaction:") {
			anchors[id] = append(json.RawMessage(nil), raw...)
		}
		t := ops[id]
		if t == nil {
			t = &TrajectoryTiming{OperationID: id}
			ops[id] = t
		}
		switch e.Type {
		case "step/start", "tool/call", "tool/ptc-dispatch-start", "compaction/start":
			if starts[id] {
				t.Invalid = true
			}
			starts[id] = true
			t.StartedAt = trajectoryTimestamp(e.Time)
		case "assistant/message", "tool/result", "tool/ptc-dispatch", "compaction/end":
			// A second distinct terminal record for one identity is ambiguous.
			if finals[id] {
				t.Invalid = true
			}
			finals[id] = true
			t.CompletedAt = trajectoryTimestamp(e.Time)
			t.Closed = true
		case "step/end":
			// Closing a failed step does not fabricate an assistant completion timestamp.
			t.Closed = true
		}
		if (e.Type == "assistant/attempt" || e.Type == "assistant/message") && t.FirstTokenAt == nil {
			t.FirstTokenAt = trajectoryStreamFirstToken(e.Data.Stream)
		}
	}
	live := make(map[string]*TrajectoryTiming)
	if a := s.assistantStream.active; a != nil {
		id := trajectoryStepID(&a.Turn, &a.Step)
		if id != "" {
			t := ops[id]
			if t == nil {
				t = &TrajectoryTiming{OperationID: id}
				ops[id] = t
			}
			if t.FirstTokenAt == nil {
				t.FirstTokenAt = trajectoryTimestamp(a.trajectoryFirstToken)
			}
			for _, rowID := range a.rows {
				live[rowID] = t
			}
		}
	}
	out := append([]Message(nil), messages...)
	mapped := make(map[string]bool)
	for i := range out {
		m := &out[i]
		m.TrajectoryTime = nil
		var e trajectoryTimeEvent
		if json.Unmarshal(m.Raw, &e) == nil {
			// Raw classifiers historically omitted EventTime for several event kinds.
			if timestamp := trajectoryTimestamp(e.Time); timestamp != nil {
				m.EventTime = *timestamp
			}
			id := trajectoryEventID(e, m.Raw)
			m.TrajectoryTime = cloneTrajectoryTiming(ops[id])
			if m.TrajectoryTime != nil {
				mapped[id] = true
			}
		}
		if t := live[m.ID]; t != nil {
			m.TrajectoryTime = cloneTrajectoryTiming(t)
		}
	}
	for id, timing := range ops {
		if !strings.HasPrefix(id, "compaction:") || mapped[id] || timing == nil {
			continue
		}
		raw := anchors[id]
		var e trajectoryTimeEvent
		_ = json.Unmarshal(raw, &e)
		eventTime := float64(0)
		if e.Time != nil {
			eventTime = *e.Time
		}
		out = append(out, Message{
			ID: id, Kind: MessageKindCompaction, Role: "system", Seq: e.Seq,
			EventType: e.Type, Raw: raw, Body: id, EventTime: eventTime,
			TrajectoryTime: cloneTrajectoryTiming(timing),
		})
	}
	return out
}
