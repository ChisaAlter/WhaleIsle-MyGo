package session

import (
	"encoding/json"
	"fmt"
	"sort"
)

// RetryAttempt is one Host-scheduled attempt, never a client retry command.
type RetryAttempt struct {
	RetryID    string `json:"retryId"`
	Turn, Step float64
	Retry      int             `json:"retry"`
	MaxRetries int             `json:"maxRetries"`
	Mode       string          `json:"mode"`
	DelayMs    float64         `json:"delayMs"`
	Failure    json.RawMessage `json:"failure"`
	Seq        float64
	State      string
}
type RetryStatus struct{ Attempts []RetryAttempt }

// projectRetryMessagesLocked uses the same durable boundaries hydrated by both
// follow and history pages. Closing the owning step/turn cancels only a pending
// wait; passing time alone never claims that a request has started.
func (s *Store) projectRetryMessagesLocked(messages []Message) []Message {
	seqs := make([]float64, 0, len(s.trajectoryBoundaries))
	for seq := range s.trajectoryBoundaries {
		seqs = append(seqs, seq)
	}
	sort.Float64s(seqs)
	rows := []Message{}
	chains := map[string]int{}
	for _, seq := range seqs {
		raw := s.trajectoryBoundaries[seq]
		var e struct {
			Type string       `json:"type"`
			Data RetryAttempt `json:"data"`
		}
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		d := e.Data
		key := fmt.Sprintf("%g:%g:%s", d.Turn, d.Step, d.RetryID)
		switch e.Type {
		case "llm/retry":
			if d.RetryID == "" || d.Retry < 1 {
				continue
			}
			d.Seq, d.State = seq, "scheduled"
			index, ok := chains[key]
			if !ok {
				index = len(rows)
				chains[key] = index
				rows = append(rows, Message{ID: "retry:" + key, Seq: seq, Kind: MessageKindText, Role: "status", EventType: "llm/retry", Raw: append(json.RawMessage(nil), raw...), Retry: &RetryStatus{}})
			}
			rows[index].Retry.Attempts = append(rows[index].Retry.Attempts, d)
		case "llm/retry-started":
			if index, ok := chains[key]; ok {
				for i := range rows[index].Retry.Attempts {
					a := &rows[index].Retry.Attempts[i]
					if a.Retry == d.Retry && a.State == "scheduled" {
						a.State = "started"
					}
				}
			}
		case "step/end", "turn/end":
			for _, row := range rows {
				attempts := row.Retry.Attempts
				a := &attempts[len(attempts)-1]
				if a.Turn == d.Turn && (e.Type == "turn/end" || a.Step == d.Step) && a.State == "scheduled" {
					a.State = "cancelled"
				}
			}
		}
	}
	out := append([]Message(nil), messages...)
	for _, row := range rows {
		index := len(out)
		for i, m := range out {
			if m.Seq > row.Seq {
				index = i
				break
			}
		}
		out = append(out, Message{})
		copy(out[index+1:], out[index:])
		out[index] = row
	}
	return out
}
