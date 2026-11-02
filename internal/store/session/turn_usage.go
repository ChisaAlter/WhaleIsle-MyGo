package session

import (
	"bytes"
	"encoding/json"
	"sort"
)

// TurnUsage is exact provider-reported accounting for a completed loaded turn.
// Optional buckets remain absent when any billed attempt did not report them.
type TurnUsage struct {
	Input, Output, Total             float64
	CacheRead, CacheWrite, Reasoning *float64
	Routes                           []string
}

type usageSample struct {
	Input      *float64 `json:"inputTokens"`
	Output     *float64 `json:"outputTokens"`
	Total      *float64 `json:"totalTokens"`
	CacheRead  *float64 `json:"cacheReadTokens"`
	CacheWrite *float64 `json:"cacheWriteTokens"`
	Reasoning  *float64 `json:"reasoningTokens"`
}

type usageEvent struct {
	Type      string `json:"type"`
	Seq, Time float64
	Data      struct {
		Turn, Step float64
		Usage      *usageSample    `json:"usage"`
		Stream     json.RawMessage `json:"stream"`
		Message    struct {
			Source struct{ Provider, Model string }
		} `json:"message"`
	} `json:"data"`
}

type usageCacheEntry struct {
	raw   string
	event usageEvent
	ok    bool
}

func lastUsage(rawStream json.RawMessage) *usageSample {
	var stream []json.RawMessage
	if len(rawStream) == 0 || bytes.Equal(bytes.TrimSpace(rawStream), []byte("null")) || json.Unmarshal(rawStream, &stream) != nil {
		return nil
	}
	var sample *usageSample
	chunks, _ := expandAssistantStream(stream)
	for _, item := range chunks {
		var chunk struct {
			Type  string       `json:"type"`
			Usage *usageSample `json:"usage"`
		}
		_ = json.Unmarshal(item.chunk, &chunk)
		if chunk.Type == "usage" {
			sample = chunk.Usage
		}
	}
	return sample
}

func exactUsage(sample *usageSample) (TurnUsage, bool) {
	if sample == nil || sample.Input == nil || sample.Output == nil {
		return TurnUsage{}, false
	}
	u := TurnUsage{Input: *sample.Input, Output: *sample.Output, CacheRead: sample.CacheRead, CacheWrite: sample.CacheWrite, Reasoning: sample.Reasoning}
	if sample.Total != nil {
		u.Total = *sample.Total
	} else {
		if sample.CacheRead == nil || sample.CacheWrite == nil {
			return TurnUsage{}, false
		}
		u.Total = u.Input + u.Output + *sample.CacheRead + *sample.CacheWrite
	}
	return u, true
}

func sumOptional(a, b *float64) *float64 {
	if a == nil || b == nil {
		return nil
	}
	v := *a + *b
	return &v
}

func addTurnUsage(total *TurnUsage, u TurnUsage, first bool) {
	if first {
		*total = u
		total.Routes = append([]string(nil), u.Routes...)
		return
	}
	total.Input += u.Input
	total.Output += u.Output
	total.Total += u.Total
	total.CacheRead = sumOptional(total.CacheRead, u.CacheRead)
	total.CacheWrite = sumOptional(total.CacheWrite, u.CacheWrite)
	total.Reasoning = sumOptional(total.Reasoning, u.Reasoning)
	if len(total.Routes) == 0 || len(u.Routes) == 0 {
		total.Routes = nil
		return
	}
	for _, route := range u.Routes {
		found := false
		for _, old := range total.Routes {
			if route == old {
				found = true
				break
			}
		}
		if !found {
			total.Routes = append(total.Routes, route)
		}
	}
}

// Turn accounting reads the durable lifecycle, including retry attempts which
// have no transcript row. Usage-only or incomplete history never becomes zero.
// trajectoryFingerprintLocked is an O(1) change detector for the builders that
// decode every historical event. Streaming appends a message (length changes)
// and grows the last raw payload, so length plus the tail identity detects
// realistic mutations without hashing the whole history on every frame.
func (s *Store) trajectoryFingerprintLocked() uint64 {
	h := uint64(1469598103934665603)
	mix := func(v uint64) { h ^= v; h *= 1099511628211 }
	// Rows that exist only while a reply streams carry no journal record, so they
	// are left out: their body grows with every chunk and would otherwise change
	// the fingerprint, and with it every cache built from the journal, each frame.
	durable := 0
	var last *Message
	for i := range s.messages {
		if volatileMessageRow(s.messages[i]) {
			continue
		}
		durable++
		last = &s.messages[i]
	}
	mix(uint64(durable))
	mix(uint64(len(s.trajectoryBoundaries)))
	if last != nil {
		mix(uint64(len(last.Raw)))
		mix(uint64(len(last.Body)))
		mix(uint64(int64(last.Seq)))
	}
	return h
}
func isUsageEventCandidate(raw json.RawMessage) bool {
	if typ, ok := peekEventType(raw); ok {
		switch typ {
		case "turn/start", "turn/end", "step/start", "step/end", "assistant/message", "assistant/attempt", "llm/retry", "llm/retry-started":
			return true
		default:
			return false
		}
	}
	if len(raw) <= 512 {
		return true
	}
	return bytes.Contains(raw, []byte("assistant/")) ||
		bytes.Contains(raw, []byte("turn/")) ||
		bytes.Contains(raw, []byte("step/")) ||
		bytes.Contains(raw, []byte("llm/retry")) ||
		bytes.Contains(raw, []byte(`\u`))
}

func (s *Store) turnUsageLocked() map[float64]TurnUsage {
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
	result := map[float64]TurnUsage{}
	var turn, step float64
	active, complete := false, false
	state := "idle"
	var sample *usageSample
	var aggregate TurnUsage
	attempts := 0
	settle := func(route string) bool {
		u, ok := exactUsage(sample)
		if !ok {
			return false
		}
		if route != "" {
			u.Routes = []string{route}
		}
		addTurnUsage(&aggregate, u, attempts == 0)
		attempts++
		return true
	}
	if s.usageEventsCache == nil {
		s.usageEventsCache = make(map[float64]usageCacheEntry, len(raws))
	}
	for _, seq := range seqs {
		raw := raws[seq]
		if !isUsageEventCandidate(raw) {
			continue
		}
		var e usageEvent
		var ok bool
		if entry, cached := s.usageEventsCache[seq]; cached && bytes.Equal([]byte(entry.raw), raw) {
			e, ok = entry.event, entry.ok
		} else {
			if json.Unmarshal(raw, &e) == nil && validRawJSONArrayOrNull(e.Data.Stream) {
				ok = true
			}
			s.usageEventsCache[seq] = usageCacheEntry{raw: string(raw), event: e, ok: ok}
		}
		if !ok {
			continue
		}
		if e.Type == "turn/start" {
			turn, active, complete, state, attempts = e.Data.Turn, true, true, "idle", 0
			aggregate = TurnUsage{}
			sample = nil
			continue
		}
		if !active {
			continue
		}
		switch e.Type {
		case "turn/end":
			if complete && e.Data.Turn == turn && state == "idle" && attempts > 0 {
				result[e.Seq] = aggregate
			}
			active = false
		case "step/start", "llm/retry-started":
			expected := "idle"
			if e.Type == "llm/retry-started" {
				expected = "retry"
			}
			if state != expected || e.Data.Turn != turn {
				complete = false
			}
			state, step, sample = "open", e.Data.Step, nil
		case "assistant/message", "assistant/attempt":
			if state != "open" || e.Data.Turn != turn || e.Data.Step != step {
				complete = false
				continue
			}
			sample = e.Data.Usage
			if sample == nil {
				sample = lastUsage(e.Data.Stream)
			}
			route := ""
			if e.Type == "assistant/message" && e.Data.Message.Source.Provider != "" && e.Data.Message.Source.Model != "" {
				route = e.Data.Message.Source.Provider + "/" + e.Data.Message.Source.Model
			}
			if !settle(route) {
				complete = false
			}
			state = "message"
			if e.Type == "assistant/attempt" {
				state = "finish"
			}
		case "llm/retry":
			if e.Data.Turn != turn || e.Data.Step != step || state == "idle" || state == "message" || state == "retry" {
				complete = false
			}
			if state == "open" && !settle("") {
				complete = false
			}
			state = "retry"
		case "step/end":
			if e.Data.Turn != turn || e.Data.Step != step || state == "idle" {
				complete = false
			}
			if state == "open" && !settle("") {
				complete = false
			}
			state = "idle"
		}
	}
	return result
}

// RecordedPerformance uses only settled rows with recorded decoding timestamps.
// The loaded-window scope is explicit; it is not a full-session estimate.
func RecordedPerformance(messages []Message) (tokens, decodeMillis float64, steps int) {
	seen := map[float64]bool{}
	for _, m := range messages {
		if m.Streaming || seen[m.Seq] || len(m.Raw) == 0 {
			continue
		}
		if m.EventType != "" && m.EventType != "assistant/message" {
			continue
		}
		if typ, ok := peekEventType(m.Raw); ok && typ != "assistant/message" {
			continue
		}
		if len(m.Raw) > 512 && !bytes.Contains(m.Raw, []byte("assistant/message")) && !bytes.Contains(m.Raw, []byte(`\u`)) {
			continue
		}
		var e usageEvent
		if json.Unmarshal(m.Raw, &e) != nil || !validRawJSONArrayOrNull(e.Data.Stream) || e.Type != "assistant/message" {
			continue
		}
		seen[m.Seq] = true
		steps++
		sample := e.Data.Usage
		if sample == nil {
			sample = lastUsage(e.Data.Stream)
		}
		t := m.TrajectoryTime
		if sample != nil && sample.Output != nil && t != nil && !t.Invalid && t.FirstTokenAt != nil && t.CompletedAt != nil && *t.CompletedAt > *t.FirstTokenAt {
			tokens += *sample.Output
			decodeMillis += *t.CompletedAt - *t.FirstTokenAt
		}
	}
	return
}
