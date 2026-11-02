package session

import (
	"bytes"
	"encoding/json"
)

type editEvent struct {
	Type      string `json:"type"`
	SurfaceOp struct {
		Op       string  `json:"op"`
		StartSeq float64 `json:"startSeq"`
		EndSeq   float64 `json:"endSeq"`
	} `json:"surfaceOp"`
	Data struct {
		Source struct {
			Kind string           `json:"kind"`
			Edit *json.RawMessage `json:"edit"`
		} `json:"source"`
		Content []struct {
			Type string `json:"type"`
		} `json:"content"`
	} `json:"data"`
}

func isHumanMessage(m Message) bool {
	if m.Role != "user" {
		return false
	}
	if !bytes.Contains(m.Raw, []byte(`"source"`)) {
		return true
	}
	var e struct {
		Data struct {
			Source struct {
				Kind string `json:"kind"`
			} `json:"source"`
		} `json:"data"`
	}
	_ = json.Unmarshal(m.Raw, &e)
	return e.Data.Source.Kind == "" || e.Data.Source.Kind == "user"
}

func annotateEditMessage(m *Message, turnSeq float64, opening *bool) {
	m.TurnSeq = turnSeq
	if isHumanMessage(*m) && *opening {
		m.TurnOpening = true
		*opening = false
	}
}

// LatestEditableMessage follows turn-opening human messages, ignoring injected context and steering.
func LatestEditableMessage(messages []Message) float64 {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if !m.Streaming && m.Seq > 0 && (m.TurnOpening || m.TurnSeq == 0) && isHumanMessage(m) {
			return m.Seq
		}
	}
	return 0
}

func CanEditMessage(s ViewState, seq float64) bool {
	if s.SelectedID == "" || s.PromptBusy || s.MessageEditBusy || seq <= 0 || LatestEditableMessage(s.Messages) != seq {
		return false
	}
	for _, m := range s.Messages {
		if m.Seq != seq || !isHumanMessage(m) {
			continue
		}
		if m.Streaming || m.Kind != MessageKindText || len(m.Files) > 0 {
			return false
		}
		var e struct {
			Data struct {
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"data"`
		}
		if len(m.Raw) > 0 && json.Unmarshal(m.Raw, &e) != nil {
			return false
		}
		for _, block := range e.Data.Content {
			if block.Type != "text" {
				return false
			}
		}
		return true
	}
	return false
}

func hasMessageEdits(messages []Message) bool {
	for _, m := range messages {
		if len(m.Raw) == 0 || !bytes.Contains(m.Raw, []byte("replace")) || !bytes.Contains(m.Raw, []byte("surfaceOp")) {
			continue
		}
		var e editEvent
		if json.Unmarshal(m.Raw, &e) == nil && e.Type == "user/message" && e.Data.Source.Kind == "user" && e.Data.Source.Edit != nil && e.SurfaceOp.Op == "replace" {
			return true
		}
	}
	return false
}

// Keep the append-only journal untouched. Reconstruct the visible chat from edit replacements on every snapshot/page.
func projectMessageEdits(messages []Message) []Message {
	hidden := map[int]bool{}
	for _, m := range messages {
		if len(m.Raw) == 0 || !bytes.Contains(m.Raw, []byte("replace")) || !bytes.Contains(m.Raw, []byte("surfaceOp")) {
			continue
		}
		var e editEvent
		if json.Unmarshal(m.Raw, &e) != nil || e.Type != "user/message" || e.Data.Source.Kind != "user" || e.Data.Source.Edit == nil || e.SurfaceOp.Op != "replace" {
			continue
		}
		replacedTurn := float64(0)
		for _, old := range messages {
			if old.Seq >= e.SurfaceOp.StartSeq && old.Seq <= e.SurfaceOp.EndSeq && old.TurnSeq > 0 {
				replacedTurn = old.TurnSeq
				break
			}
		}
		for i, old := range messages {
			if old.Seq < m.Seq && ((old.Seq >= e.SurfaceOp.StartSeq && old.Seq <= e.SurfaceOp.EndSeq) || (replacedTurn > 0 && old.TurnSeq == replacedTurn)) {
				hidden[i] = true
			}
		}
	}
	out := make([]Message, 0, len(messages))
	for i, m := range messages {
		if !hidden[i] {
			out = append(out, m)
		}
	}
	return out
}
