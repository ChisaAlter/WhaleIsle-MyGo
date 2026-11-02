package session

import (
	"bytes"
	"encoding/json"
	"math"
)

type AgentTiming struct {
	SettledMs      float64
	Since, Through *float64
}

// Immutable projection values are safe to share with a pane snapshot. Each key
// has its own watermark: token and timing frames can arrive independently.
type AgentMetrics struct {
	Tokens              *float64
	Timing              *AgentTiming
	Completed           bool
	Ready               bool
	TokenSeq, TimingSeq float64
}

func nonnegative(v *float64) bool {
	return v != nil && !math.IsNaN(*v) && !math.IsInf(*v, 0) && *v >= 0
}
func decodeAgentMetrics(raw json.RawMessage, seq float64) AgentMetrics {
	m := AgentMetrics{Ready: true, TokenSeq: seq, TimingSeq: seq}
	if !bytes.Contains(raw, []byte(`"tokenUsage"`)) && !bytes.Contains(raw, []byte(`"subagentTiming"`)) {
		return m
	}
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return m
	}
	var usage struct {
		Uncached *float64 `json:"uncachedInputTokens"`
		Output   *float64 `json:"outputTokens"`
		Read     *float64 `json:"cacheReadTokens"`
		Write    *float64 `json:"cacheWriteTokens"`
	}
	if json.Unmarshal(values["tokenUsage"], &usage) == nil && nonnegative(usage.Uncached) && nonnegative(usage.Output) && nonnegative(usage.Read) && nonnegative(usage.Write) {
		total := *usage.Uncached + *usage.Output + *usage.Read + *usage.Write
		if !math.IsInf(total, 0) {
			m.Tokens = &total
		}
	}
	var timing struct {
		Settled *float64 `json:"settledMs"`
		Active  *struct {
			Since   *float64 `json:"since"`
			Through *float64 `json:"through"`
		} `json:"active"`
		Completed bool `json:"lastTurnCompleted"`
	}
	if json.Unmarshal(values["subagentTiming"], &timing) == nil && nonnegative(timing.Settled) {
		if timing.Active == nil {
			m.Timing = &AgentTiming{SettledMs: *timing.Settled}
		} else if nonnegative(timing.Active.Since) && nonnegative(timing.Active.Through) {
			m.Timing = &AgentTiming{SettledMs: *timing.Settled, Since: timing.Active.Since, Through: timing.Active.Through}
		}
		if m.Timing != nil {
			m.Completed = timing.Completed
		}
	}
	return m
}
func (m AgentMetrics) Duration(running bool, now float64) (float64, bool) {
	if m.Timing == nil {
		return 0, false
	}
	t := m.Timing
	duration := t.SettledMs
	if t.Since != nil {
		end := *t.Through
		if running {
			end = now
		}
		duration += math.Max(0, end-*t.Since)
	}
	return duration, true
}
func (s *Store) applyAgentMetricsControlLocked(raw json.RawMessage) {
	s.invalidateViewLocked()
	var frame struct {
		Type, SessionID, Key string
		Seq                  *float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	apply := func(id string, seq float64, values json.RawMessage, key string) {
		if math.IsNaN(seq) || math.IsInf(seq, 0) || seq < -1 || math.Trunc(seq) != seq {
			return
		}
		next := decodeAgentMetrics(values, seq)
		for i := range s.items {
			if s.items[i].SessionID == id {
				m := &s.items[i].AgentMetrics
				if (key == "" || key == "tokenUsage") && (!m.Ready || seq >= m.TokenSeq) {
					m.Tokens = next.Tokens
					m.TokenSeq = seq
				}
				if (key == "" || key == "subagentTiming") && (!m.Ready || seq >= m.TimingSeq) {
					m.Timing = next.Timing
					m.Completed = next.Completed
					m.TimingSeq = seq
				}
				m.Ready = true
				break
			}
		}
	}
	if frame.Type == "projection" && frame.Seq != nil && (frame.Key == "tokenUsage" || frame.Key == "subagentTiming") {
		values, _ := json.Marshal(map[string]json.RawMessage{frame.Key: frame.Value})
		apply(frame.SessionID, *frame.Seq, values, frame.Key)
	}
	if frame.Type == "baseline" {
		var b struct {
			Projections map[string]agentNavigationProjection
		}
		if json.Unmarshal(frame.Value, &b) == nil {
			for id, p := range b.Projections {
				if p.AsOfSeq != nil {
					apply(id, *p.AsOfSeq, p.Values, "")
				}
			}
		}
	}
}

func mergeAgentMetrics(next, old AgentMetrics) AgentMetrics {
	if !next.Ready {
		return old
	}
	if old.Ready && old.TokenSeq > next.TokenSeq {
		next.Tokens = old.Tokens
		next.TokenSeq = old.TokenSeq
	}
	if old.Ready && old.TimingSeq > next.TimingSeq {
		next.Timing = old.Timing
		next.Completed = old.Completed
		next.TimingSeq = old.TimingSeq
	}
	return next
}
